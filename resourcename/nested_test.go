package resourcename

import "testing"

type Address struct {
	City string `resource:"city"`
	Zip  string `resource:"zip"`
}

type UserWithAddress struct {
	_       struct{} `resource:"//bobthebuilder.com/users/{id}/{address.city}/{address.zip}"`
	ID      string   `resource:"id"`
	Address Address  `resource:"address"`
}

type Location struct {
	Country string `resource:"country"`
	Region  string `resource:"region"`
}

type Company struct {
	Name     string   `resource:"name"`
	Location Location `resource:"location"`
}

type Employee struct {
	_       struct{} `resource:"//bobthebuilder.com/employees/{id}/{company.name}/{company.location.country}"`
	ID      string   `resource:"id"`
	Company Company  `resource:"company"`
}

func TestNestedStructMarshal(t *testing.T) {
	u := &UserWithAddress{ID: "u42", Address: Address{City: "NYC", Zip: "10001"}}
	rn, err := MarshalResource(u)
	if err != nil {
		t.Fatalf("MarshalResource() error = %v", err)
	}
	expected := "//bobthebuilder.com/users/u42/NYC/10001"
	if rn != expected {
		t.Errorf("got %v, want %v", rn, expected)
	}

	e := &Employee{ID: "e100", Company: Company{Name: "Acme", Location: Location{Country: "USA", Region: "West"}}}
	rn, err = MarshalResource(e)
	if err != nil {
		t.Fatalf("MarshalResource() error = %v", err)
	}
	expected = "//bobthebuilder.com/employees/e100/Acme/USA"
	if rn != expected {
		t.Errorf("got %v, want %v", rn, expected)
	}
}

func TestNestedStructUnmarshal(t *testing.T) {
	u := &UserWithAddress{}
	err := UnmarshalResource("//bobthebuilder.com/users/u42/NYC/10001", u)
	if err != nil {
		t.Fatalf("UnmarshalResource() error = %v", err)
	}
	if u.ID != "u42" || u.Address.City != "NYC" || u.Address.Zip != "10001" {
		t.Errorf("got ID=%v, City=%v, Zip=%v", u.ID, u.Address.City, u.Address.Zip)
	}

	e := &Employee{}
	err = UnmarshalResource("//bobthebuilder.com/employees/e100/Acme/USA", e)
	if err != nil {
		t.Fatalf("UnmarshalResource() error = %v", err)
	}
	if e.ID != "e100" || e.Company.Name != "Acme" || e.Company.Location.Country != "USA" {
		t.Errorf("got ID=%v, Name=%v, Country=%v", e.ID, e.Company.Name, e.Company.Location.Country)
	}
}

func TestNestedStructRoundTrip(t *testing.T) {
	u := &UserWithAddress{ID: "u42", Address: Address{City: "SF", Zip: "94102"}}
	rn, _ := MarshalResource(u)
	u2 := &UserWithAddress{}
	if err := UnmarshalResource(rn, u2); err != nil {
		t.Errorf("unmarshal failed: %v", err)
	}
	rn2, _ := MarshalResource(u2)
	if rn != rn2 {
		t.Errorf("round trip failed: %v != %v", rn, rn2)
	}

	e := &Employee{ID: "e200", Company: Company{Name: "Tech", Location: Location{Country: "CA", Region: "E"}}}
	rn, _ = MarshalResource(e)
	e2 := &Employee{}
	if err := UnmarshalResource(rn, e2); err != nil {
		t.Errorf("unmarshal failed: %v", err)
	}
	rn2, _ = MarshalResource(e2)
	if rn != rn2 {
		t.Errorf("round trip failed: %v != %v", rn, rn2)
	}
}
