# Milestone M1: Foundation (Walking Skeleton)

| Field      | Value |
|------------|-------|
| Status     | ✅ IN_PROGRESS |
| Tests      | 64 |
| Packages   | 11 |

## Goal

Scaffolding of the project:

- Project structure
- Makefile
- Dockerfiles
- Helm chart
- CI/CD pipeline (github actions)
- Documentation

## Deliverable

Initialized git repository with the following:

- Project structure
- Makefile
- Dockerfiles
- Helm chart
- CI/CD pipeline (github actions)
- Documentation

A Makefile to:

- Build the project
- Run tests: indipendent tasks for unit, integration using testcontainers, benchmark and race.
- Create the docker image

An helm chart to deploy the required infrastructural dependencies, using the following services:

- MinIO instance
- Redis instance
- Postgresql instance
- Nats instance
- Qdrant instance

A GitHub Actions workflow to:

- Run tests
- Dependabot version updates

Documentation: 

- Project README with getting started and build instructions
- Chart README with getting started and deploy instructions

## Specs Covered

| Spec ID | Title | Status |
|---------|-------|--------|
| SPEC-FR-M1.1 | Build System & Layout | VERIFIED |
| SPEC-FR-M1.2 | Containerization | VERIFIED |
| SPEC-FR-M1.3 | Infrastructure Deployment | IN_PROGRESS |
| SPEC-FR-M1.4 | Continuous Integration | IN_PROGRESS |
| SPEC-FR-M1.5 | Project Documentation | VERIFIED |
