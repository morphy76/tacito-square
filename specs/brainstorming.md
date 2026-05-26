# Brainstorming

- Tacito multistage builds addirional make steps, they are faster than the single stage builds and decouples the workspace from tooling and runtime dependencies. This is especially useful for development, where we want to iterate quickly on the workspace without having to rebuild the entire image.
- Summary agent to reduce conversations when they start to get long. This can be implemented as a separate tool that the agent can call when the conversation history exceeds a certain threshold. The summary agent can use techniques like extractive summarization or abstractive summarization to generate a concise summary of the conversation history, which can then be used as context for future interactions. This can help improve the performance and relevance of the agent's responses while keeping the conversation manageable.
- Gateway agent for user interaction streaming reasoning and answers outbound and receiving inbound. This agent can act as a central hub for managing user interactions, allowing for real-time streaming of reasoning processes and answers. It can handle incoming requests from users, process them using the appropriate agents or tools, and then stream the results back to the users in a timely manner. This can enhance the user experience by providing faster responses and enabling more dynamic interactions with the system.
- E2E automated functional tests to validate the entire system, including the UI, the API, and the K8s operator. These tests should be written in a way that they can be run as part of the CI/CD pipeline and should be able to be run locally by developers. They should also be able to be run in a Docker container and should be able to be run on a Kubernetes cluster. The tests should be written in a way that they can be run in parallel and should be able to be run in a non-blocking manner. The tests should be written in a way that they can be run in a parallel manner and should be able to be run in a non-blocking manner.
- MCP discovery when mcp added to agent to limit the agent to certain tools
- Skillsets to aggregate multiple skills
- Online skillsets, e.g. from github
- how RAG and websearch fit into the picture
- Roles and functional ownership, in particular within the tenant and quota-bound (platform QoS)
- postman collection for keeper -> or any test suite
- hardcoded Version vs VERSION files for component lifecycle
- BFF & FE rules, best practices, no http session, api tokens in encrypted cookies
- coverage of integration and benchmark tests
