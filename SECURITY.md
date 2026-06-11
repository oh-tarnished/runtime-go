# Security Policy

## Supported Versions

Only the latest release of each module receives security fixes.

| Module | Supported |
| ------ | --------- |
| ulid | latest |
| resourcename | latest |
| config | latest |
| network | latest |
| grpc | latest |
| system | latest |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Use GitHub's private vulnerability reporting:
**[Report a vulnerability →](https://github.com/oh-tarnished/runtime-go/security/advisories/new)**

This creates a private advisory visible only to you and the maintainers — nothing is public until a fix is released.

Include:
- The module and version affected
- A description of the vulnerability and its potential impact
- Steps to reproduce or a minimal proof-of-concept (if available)
- Any suggested mitigations

You will receive an acknowledgement within **72 hours**.

## Disclosure Timeline

| Step | Target |
| ---- | ------ |
| Acknowledgement | ≤ 72 hours |
| Triage and initial assessment | ≤ 7 days |
| Fix published and release tagged | ≤ 30 days |
| Public disclosure (CVE if applicable) | after fix is released |

## Scope

This policy covers all Go code in the `github.com/oh-tarnished/runtime-go/*` modules.

Out of scope:
- Vulnerabilities in third-party dependencies (report to the upstream project)
- Issues already fixed in the latest release
- Theoretical vulnerabilities with no practical exploit

## Known Considerations

### `system` module
The `system` module executes OS-level commands (`shutdown`, `loginctl`) and is intentionally privileged. It should only be imported by services running with appropriate Linux capabilities. Do not expose its functions to untrusted callers.

### `network` module
The HTTP and WebSocket clients follow redirects and trust the server's TLS certificate using the system root CA pool. Custom CA pinning is not built in — callers that need it should configure a custom `http.Transport` before passing it to the client.

### `config` module
Environment variable loading (`LoadEnv`) strips the configured prefix but does not sanitize key names. Ensure the prefix is set correctly to avoid unintended variable injection from the process environment.
