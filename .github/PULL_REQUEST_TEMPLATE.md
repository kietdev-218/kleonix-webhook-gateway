## Description

Please include a clear and concise summary of the change and which issue is fixed. Include relevant motivation, context, and any structural design choices made.

Fixes # (issue)

## Type of change

Please delete options that are not relevant.

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] ♻️ Refactoring (Code cleanup, structural changes without feature impact)
- [ ] 📚 Documentation update

## How Has This Been Tested?

Please describe the tests that you ran to verify your changes. 
- [ ] Added/Updated Go Unit Tests (`go test -v ./...`)
- [ ] Tested locally with Docker Compose and mocked Kratos webhooks
- [ ] Verified Prometheus metrics / health endpoints work as expected

## Checklist:

- [ ] My code follows the Clean Architecture and SOLID principles established in this project.
- [ ] I have performed a self-review of my own code.
- [ ] I have commented my code, particularly in hard-to-understand areas.
- [ ] I have made corresponding changes to the documentation (`README.md`, `.env.example`).
- [ ] New and existing unit tests pass locally with my changes.
- [ ] `golangci-lint` passes locally without any warnings.
