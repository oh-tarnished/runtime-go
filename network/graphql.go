// GraphQL client implementation.
//
// This file provides the GraphQL client used when ClientType is GraphQLConnClient.
// It supports queries, mutations (typed and raw), and optional connectivity
// verification via a configurable introspection query.
package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/oh-tarnished/runtime-go/network/shared"
)

// DefaultGraphQLConnectivityQuery is the default query sent to verify the GraphQL
// server is reachable during Connect/Reconnect. Override with
// ConnectionOptions.GraphQLConnectivityQuery for strict servers that limit introspection.
const DefaultGraphQLConnectivityQuery = `query { __typename }`

// GraphQL scalar type aliases: use network.Boolean, network.String, network.ID, etc.
// in variables and struct tags without importing the underlying graphql package.
type (
	// Boolean represents true or false values for GraphQL.
	Boolean = graphql.Boolean

	// Float represents signed double-precision fractional values as specified by IEEE 754.
	Float = graphql.Float

	// Int represents non-fractional signed whole numeric values.
	// Int can represent values between -(2^31) and 2^31 - 1.
	Int = graphql.Int

	// String represents textual data as UTF-8 character sequences.
	// This type is most often used by GraphQL to represent free-form human-readable text.
	String = graphql.String

	// ID represents a unique identifier that is Base64 obfuscated.
	// It is often used to refetch an object or as key for a cache.
	ID = graphql.ID
)

// GraphQLClient is a GraphQL API client. It embeds ConnectionOptions and provides
// Connect, Reconnect, Close, Query, Mutation, MutationWithInput, ExecuteRawQuery,
// and ExecRawMutation. Create via NewConnection(GraphQLConnClient) and AsGraphQLConnectionType.
type GraphQLClient struct {
	client            *graphql.Client
	ConnectionOptions // URL, Timeout, Headers, SkipConnectivityCheck, GraphQLConnectivityQuery
}

// GraphQLResult is the result of an asynchronous GraphQL operation. Response holds
// the decoded result; Error is non-nil if the operation failed.
type GraphQLResult struct {
	Response interface{} // Filled query/mutation struct or raw response map
	Error    error       // Non-nil if the operation failed
}

// Connect configures the GraphQL client and optionally verifies server reachability.
// If opts.Timeout <= 0, DefaultTimeout is used. If SkipConnectivityCheck is true,
// no connectivity query is sent; otherwise the connectivity query is run and
// Connect returns an error on failure.
func (g *GraphQLClient) Connect(opts ConnectionOptions) error {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
		shared.Pulse.Logger.Debug("GraphQL Connect using default timeout", "timeout", opts.Timeout)
	}
	shared.Pulse.Logger.Debug("GraphQL Connect called", "host", opts.URL.Host, "timeout", opts.Timeout)
	g.ConnectionOptions = opts

	// Build the full URL for the GraphQL endpoint
	fullURL, err := buildFullURL(opts.URL, 0)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to build GraphQL URL", "error", err, "urlOptions", opts.URL)
		return fmt.Errorf("failed to build full URL: %w", err)
	}
	shared.Pulse.Logger.Debug("GraphQL URL built", "fullURL", fullURL)

	// Initialize the GraphQL client with the full URL
	g.client = graphql.NewClient(fullURL, &http.Client{Timeout: opts.Timeout})
	shared.Pulse.Logger.Debug("GraphQL client initialized", "timeout", opts.Timeout)

	if opts.SkipConnectivityCheck {
		shared.Pulse.Logger.Infof("GraphQL client configured (connectivity check skipped) url=%s host=%s", fullURL, opts.URL.Host)
		return nil
	}

	connectivityQuery := DefaultGraphQLConnectivityQuery
	if opts.GraphQLConnectivityQuery != "" {
		connectivityQuery = opts.GraphQLConnectivityQuery
		shared.Pulse.Logger.Debug("Using custom GraphQL connectivity query")
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	if _, err := g.client.ExecRaw(ctx, connectivityQuery, nil); err != nil {
		shared.Pulse.Logger.Error("Failed to verify GraphQL connection to host", "error", err, "url", fullURL, "host", opts.URL.Host)
		return fmt.Errorf("failed to connect to GraphQL server at %s: %w", opts.URL.Host, err)
	}
	shared.Pulse.Logger.Infof("Connected to GraphQL server successfully url=%s host=%s headers=%d", fullURL, opts.URL.Host, len(g.Headers))
	return nil
}

// Reconnect tears down the current client and re-establishes it with the same
// options (URL, Timeout, SkipConnectivityCheck, GraphQLConnectivityQuery). If
// SkipConnectivityCheck is false, the connectivity query is run again. Returns
// an error if the client was never initialized or if reconnection fails.
func (g *GraphQLClient) Reconnect() error {
	shared.Pulse.Logger.Debug("GraphQL Reconnect called")
	if g.client == nil {
		shared.Pulse.Logger.Error("GraphQL client is not initialized for reconnect")
		return fmt.Errorf("GraphQL client is not initialized")
	}

	// Build the full URL for the GraphQL endpoint
	fullURL, err := buildFullURL(g.URL, 0)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to build URL for reconnect", "error", err)
		return fmt.Errorf("failed to build full URL: %w", err)
	}

	timeout := g.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	g.client = graphql.NewClient(fullURL, &http.Client{Timeout: timeout})
	if g.SkipConnectivityCheck {
		shared.Pulse.Logger.Infof("GraphQL client reconfigured (connectivity check skipped) url=%s host=%s", fullURL, g.URL.Host)
		return nil
	}
	connectivityQuery := DefaultGraphQLConnectivityQuery
	if g.GraphQLConnectivityQuery != "" {
		connectivityQuery = g.GraphQLConnectivityQuery
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if _, err := g.client.ExecRaw(ctx, connectivityQuery, nil); err != nil {
		shared.Pulse.Logger.Error("Failed to verify GraphQL reconnection to host", "error", err, "url", fullURL, "host", g.URL.Host)
		return fmt.Errorf("failed to reconnect to GraphQL server at %s: %w", g.URL.Host, err)
	}
	shared.Pulse.Logger.Infof("Reconnected to GraphQL server successfully url=%s host=%s", fullURL, g.URL.Host)
	return nil
}

// Close clears the GraphQL client and releases resources. The client is no longer
// usable until Connect is called again.
func (g *GraphQLClient) Close() error {
	host := g.URL.Host
	g.client = nil
	shared.Pulse.Logger.Debug("Closing GraphQL connection", "host", host)
	return nil
}

// Query runs a GraphQL query asynchronously. The query argument is a struct whose
// graphql struct tags define the query shape; it is filled with the response on
// success. Variables may be nil. The returned channel receives exactly one
// GraphQLResult (Response set to the query struct, or Error set) and is then closed.
// The operation uses the client's Timeout.
func (g *GraphQLClient) Query(query interface{}, variables map[string]interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)

		if g.client == nil {
			shared.Pulse.Logger.Error("GraphQL client is not initialized")
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		shared.Pulse.Logger.Debug("Executing GraphQL query", "variables", variables)

		err := g.client.Query(ctx, query, variables)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to execute GraphQL query", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute query: %w", err)}
			return
		}

		shared.Pulse.Logger.Debug("GraphQL query executed successfully")
		resultChan <- GraphQLResult{Response: query}
	}()
	return resultChan
}

// Mutation runs a GraphQL mutation asynchronously. The mutation argument is a struct
// with graphql tags defining the mutation; it is filled with the response on success.
// Variables may be nil. The returned channel receives one GraphQLResult and closes.
func (g *GraphQLClient) Mutation(mutation any, variables map[string]interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)

		if g.client == nil {
			shared.Pulse.Logger.Error("GraphQL client is not initialized")
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		shared.Pulse.Logger.Debug("Executing GraphQL mutation", "variables", variables)

		err := g.client.Mutate(ctx, mutation, variables)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to execute GraphQL mutation", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute mutation: %w", err)}
			return
		}

		shared.Pulse.Logger.Debug("GraphQL mutation executed successfully")
		resultChan <- GraphQLResult{Response: mutation}
	}()
	return resultChan
}

// MutationWithInput runs a GraphQL mutation by building arguments from a struct
// with json tags. mutationName is the mutation field name (e.g. "createUser");
// input is the argument struct; response is a struct pointer that receives the
// mutation result. The mutation is sent as mutationName(input: $input) with
// variables derived from input. Returns a channel that receives one GraphQLResult.
//
// Example:
//
//	input := CreateUserInput{Name: "John", Email: "john@example.com"}
//	var response struct { CreateUser struct { ID, Name string } }
//	result := <-client.MutationWithInput("createUser", input, &response)
func (g *GraphQLClient) MutationWithInput(mutationName string, input interface{}, response interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)

		if g.client == nil {
			shared.Pulse.Logger.Error("GraphQL client is not initialized")
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}

		// Convert input struct to map using json tags
		variables, err := StructToMap(input)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to convert input struct to map", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to convert input: %w", err)}
			return
		}

		// Build the mutation arguments string
		args := BuildGraphQLArgs(variables)

		// Build the mutation struct dynamically
		mutationStruct := buildDynamicMutation(mutationName, args, response)

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		shared.Pulse.Logger.Debug("Executing GraphQL mutation with input", "mutation", mutationName, "args", args)

		err = g.client.Mutate(ctx, mutationStruct, nil)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to execute GraphQL mutation", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute mutation: %w", err)}
			return
		}

		shared.Pulse.Logger.Debug("GraphQL mutation executed successfully")
		resultChan <- GraphQLResult{Response: mutationStruct}
	}()
	return resultChan
}

// ExecuteRawQuery sends a raw GraphQL query (or mutation) string with optional
// variables and returns the response as a map[string]interface{}. The returned
// channel receives one GraphQLResult; Response is the parsed JSON response map.
func (g *GraphQLClient) ExecuteRawQuery(query string, variables map[string]interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)

		if g.client == nil {
			shared.Pulse.Logger.Error("GraphQL client is not initialized")
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		shared.Pulse.Logger.Debug("Executing raw GraphQL query", "query", query, "variables", variables)

		var response map[string]interface{}
		responsebytes, err := g.client.ExecRaw(ctx, query, variables)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to execute raw GraphQL query", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute raw query: %w", err)}
			return
		}
		err = json.Unmarshal(responsebytes, &response)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to unmarshal raw GraphQL response", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to unmarshal raw response: %w", err)}
			return
		}
		shared.Pulse.Logger.Debug("Raw GraphQL query executed successfully")
		resultChan <- GraphQLResult{Response: response}
	}()
	return resultChan
}

// ExecRawMutation sends a raw GraphQL mutation string (mutation must be string-typed
// at runtime) with optional variables and returns the response as a map. The
// channel receives one GraphQLResult with Response as the parsed JSON map.
func (g *GraphQLClient) ExecRawMutation(mutation any, variables map[string]interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)

		if g.client == nil {
			shared.Pulse.Logger.Error("GraphQL client is not initialized")
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		shared.Pulse.Logger.Debug("Executing raw GraphQL mutation", "variables", variables)

		var response map[string]interface{}
		responsebytes, err := g.client.MutateRaw(ctx, mutation.(string), variables)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to execute raw GraphQL mutation", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute raw mutation: %w", err)}
			return
		}
		err = json.Unmarshal(responsebytes, &response)
		if err != nil {
			shared.Pulse.Logger.Error("Failed to unmarshal raw GraphQL mutation response", "error", err)
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to unmarshal raw mutation response: %w", err)}
			return
		}
		shared.Pulse.Logger.Debug("Raw GraphQL mutation executed successfully")
		resultChan <- GraphQLResult{Response: response}
	}()
	return resultChan
}

// StructToMap converts a struct with json tags into a map[string]interface{}
// via JSON marshal/unmarshal. Useful for building GraphQL variables from structs.
func StructToMap(input interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// BuildGraphQLArgs formats a map of variables as a GraphQL arguments string
// (e.g. "id: \"1\", name: \"foo\"") for use in inline mutation/query strings.
func BuildGraphQLArgs(variables map[string]interface{}) string {
	args := make([]string, 0, len(variables))
	for key, value := range variables {
		var argValue string
		switch v := value.(type) {
		case string:
			argValue = fmt.Sprintf("%s: %q", key, v)
		case int, int64, float64, bool:
			argValue = fmt.Sprintf("%s: %v", key, v)
		default:
			// For complex types, marshal to JSON
			jsonBytes, _ := json.Marshal(v)
			argValue = fmt.Sprintf("%s: %s", key, string(jsonBytes))
		}
		args = append(args, argValue)
	}
	return strings.Join(args, ", ")
}

// buildDynamicMutation builds a struct type with a single graphql-tagged field
// for the given mutation name and argument string, and returns a pointer to an
// instance of that struct (used by MutationWithInput).
func buildDynamicMutation(mutationName string, args string, response interface{}) interface{} {
	// Get the response type
	responseVal := reflect.ValueOf(response)
	if responseVal.Kind() != reflect.Ptr {
		// Response must be a pointer
		return nil
	}
	responseType := responseVal.Elem().Type()

	// Create the graphql tag with the mutation name and arguments
	graphqlTag := fmt.Sprintf(`graphql:"%s(%s)"`, mutationName, args)

	// Capitalize first letter of mutation name for the field name
	fieldName := strings.ToUpper(mutationName[:1]) + mutationName[1:]

	// Create a struct field with the response type
	fields := []reflect.StructField{
		{
			Name: fieldName,
			Type: responseType,
			Tag:  reflect.StructTag(graphqlTag),
		},
	}

	// Create the struct type and instantiate it
	structType := reflect.StructOf(fields)
	mutationStruct := reflect.New(structType)

	return mutationStruct.Interface()
}
