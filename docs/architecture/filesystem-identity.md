# Identidade de filesystem

Paths exibidos ao usuário, paths canônicos e identidade física do filesystem
são responsabilidades diferentes. Em aliases como `/var` e `/private/var`, o
Nexus deve usar a identidade apropriada para comparar objetos sem perder uma
representação estável para UI e API.
