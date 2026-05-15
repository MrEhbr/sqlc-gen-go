# Changelog

All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

---
## [0.3.1](https://github.com/MrEhbr/sqlc-gen-go/compare/v0.3.0..v0.3.1) - 2026-05-15

### Bug Fixes

- **(templates)** restore sqlc.embed scan expansion in all drivers - ([3a76c5d](https://github.com/MrEhbr/sqlc-gen-go/commit/3a76c5d1500cfbc562a093be78f91dcc15a04c8f)) - Aleksei Burmistrov

### Refactoring

- **(types)** simplify enum name lookup with early returns - ([e22d80c](https://github.com/MrEhbr/sqlc-gen-go/commit/e22d80c60231a3dd05e3b60862269a2d8fd1b50d)) - Aleksei Burmistrov

### Tests

- **(examples)** add sqlc.embed roundtrip tests across all drivers - ([040e9f4](https://github.com/MrEhbr/sqlc-gen-go/commit/040e9f4e71e72b8335bd17d60d8181af4234c979)) - Aleksei Burmistrov

---
## [0.3.0](https://github.com/MrEhbr/sqlc-gen-go/compare/v0.2.0..v0.3.0) - 2025-12-16

### Features

- add WithTx to QueryExecutor interface - ([f734d1e](https://github.com/MrEhbr/sqlc-gen-go/commit/f734d1e6ee9bf48aebf1c692b2efb11912bf090f)) - Aleksei Burmistrov

---
## [0.2.0](https://github.com/MrEhbr/sqlc-gen-go/compare/v0.1.2..v0.2.0) - 2025-12-15

### Bug Fixes

- **(templates)** support scalar returns in query templates - ([63a1812](https://github.com/MrEhbr/sqlc-gen-go/commit/63a1812f2e96a59a8b1878d9ced1a7257e46662b)) - Aleksei Burmistrov

### Features

- add mock executor support for queries - ([feb3db0](https://github.com/MrEhbr/sqlc-gen-go/commit/feb3db0b1ee62caff00afeccee982cdcf28fb897)) - Aleksei Burmistrov

---
## [0.1.2](https://github.com/MrEhbr/sqlc-gen-go/compare/v0.1.1..v0.1.2) - 2025-10-13

### Miscellaneous Chores

- refine release automation to ignore changelog-related changes - ([0e4bc0f](https://github.com/MrEhbr/sqlc-gen-go/commit/0e4bc0f662f85bd72420d0499a6b9d0eea552b29)) - Aleksei Burmistrov

---
## [0.1.1](https://github.com/MrEhbr/sqlc-gen-go/compare/v0.1.0..v0.1.1) - 2025-10-13

### Bug Fixes

- **(release)** reorder workflow steps to fix empty changelog generation - ([8edf1c4](https://github.com/MrEhbr/sqlc-gen-go/commit/8edf1c43b86f8d5735ab88fc4a4c5cd4aa4cf522)) - Aleksei Burmistrov

### CI/CD

- optimize workflow triggers with path-based filtering - ([7977058](https://github.com/MrEhbr/sqlc-gen-go/commit/7977058784bc2c8ee3e6d9f9d9a690cab742f39e)) - Aleksei Burmistrov

---
## [0.1.0] - 2025-10-13

### Bug Fixes

- **(gen)** apply query_parameter_limit to generate individual params instead of structs - ([c865f30](https://github.com/MrEhbr/sqlc-gen-go/commit/c865f3062812746a24a306d16a0a089bbeebb1d5)) - Aleksei Burmistrov

### CI/CD

- implement comprehensive CI/CD pipeline with release automation - ([38e1968](https://github.com/MrEhbr/sqlc-gen-go/commit/38e19680655ed1199567fe92c0fa182467e04b1d)) - Aleksei Burmistrov

### Documentation

- Update README to explain building from source and migrating - ([3fe89b8](https://github.com/MrEhbr/sqlc-gen-go/commit/3fe89b8062caada827d9241329ce6800af3f55f1)) - Andrew Benton
- Update README with new URLs and SHAs for 1.0.1 release - ([40cac71](https://github.com/MrEhbr/sqlc-gen-go/commit/40cac7122dada30442c74c8840ddd7b3f0acc18f)) - Andrew Benton
- Update README with instructions for 1.1.0 - ([8af8f79](https://github.com/MrEhbr/sqlc-gen-go/commit/8af8f7964d140bd154d08f16088f3f1b0b9bc99c)) - Kyle Conroy
- update README with query struct pattern and new configuration options - ([cebefde](https://github.com/MrEhbr/sqlc-gen-go/commit/cebefde93d0b12d370374954d71aa3154302a4cc)) - Aleksei Burmistrov
- add plugin installation and SHA256 checksum usage instructions - ([00312f9](https://github.com/MrEhbr/sqlc-gen-go/commit/00312f9c4a11e5b532087fd28e274e4590ff2977)) - Aleksei Burmistrov
- revamp README with comprehensive improvements and cleanup - ([a74982c](https://github.com/MrEhbr/sqlc-gen-go/commit/a74982c3666eda0624e7a0dc4d752411a70dfb0d)) - Aleksei Burmistrov

### Features

- add support for split package generation and query struct pattern options - ([eabd493](https://github.com/MrEhbr/sqlc-gen-go/commit/eabd493ec2c1c89e07c496238ff2453cd862cfb6)) - Aleksei Burmistrov
- replace interface-based queries with query struct pattern in all templates - ([fc0b5ae](https://github.com/MrEhbr/sqlc-gen-go/commit/fc0b5ae689f0a07c93d95efdf2be267a81a0da7f)) - Aleksei Burmistrov

### Miscellaneous Chores

- apply code formatting and update dependencies - ([3ec3600](https://github.com/MrEhbr/sqlc-gen-go/commit/3ec360039c21e50d913e3eb8521ad6fe4fd3fda1)) - Aleksei Burmistrov

### Tests

- migrate example tests to testcontainers-go - ([c307c0a](https://github.com/MrEhbr/sqlc-gen-go/commit/c307c0aac538dfda21143c8a765853cf4506cf24)) - Aleksei Burmistrov

### Build

- add Nix flake and improve build tooling configuration - ([3db41dc](https://github.com/MrEhbr/sqlc-gen-go/commit/3db41dcfc0817dd8b37351a668a524183037c168)) - Aleksei Burmistrov
- update Go dependencies to latest versions - ([5f6c2d3](https://github.com/MrEhbr/sqlc-gen-go/commit/5f6c2d3babd8ebb043ced695b82d086c82bce419)) - Aleksei Burmistrov

<!-- generated by git-cliff -->
