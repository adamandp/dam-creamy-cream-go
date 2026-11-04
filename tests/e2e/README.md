# End-to-end testing

Use this directory for application-wide HTTP acceptance tests. Test against an isolated PostgreSQL database and disposable Redis/MinIO resources.

Unit tests are in tests/unit and integration tests are in tests/integration. CI starts separate services for the integration suite.
