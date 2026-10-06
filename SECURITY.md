# Security policy

This is an upstream project and is not affiliated with any Red Hat product.

The program collects system heartbeats and sends them to a Prometheus Remote Write endpoint.

## Reporting a vulnerability

Report security vulnerabilities privately through [GitHub Security Advisories](https://github.com/m-horky/rhc-heartbeat/security/advisories/new). Include the affected commit, the impact, and reproducible steps.

## Disclaimer

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

---

# Threat model

## Security-relevant behavior

The following points are known and expected features, and should not be considered security findings.

- TLS verification and HTTP-only connections can be set in configuration. They are meant to be used for development and on trusted networks.
- Paths read by the program can be configured with environment variables, which are not attacker-controllable. Customizable paths are only supported for development and testing.
- `make server` starts a development-only Prometheus receiver and is not included in product packages.
