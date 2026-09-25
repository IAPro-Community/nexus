# IAPro Nexus

> **Estado de release:** el candidato de source es `1.0.0-beta.2` e integra
> Linux amd64, Windows amd64 y macOS arm64. Todavía **no existe una GitHub
> Release pública**; la tag histórica `v1.0.0-beta.1` no equivale a una
> distribución binaria publicada.

**Una estación de trabajo local para operar coding agents.**

Proyectos, agentes persistentes, sesiones de terminal supervisadas, proveedores, worktrees y automatización en una experiencia integrada de Web, Desktop y CLI.

<p align="center"><a href="README.md">Português</a> · <a href="README.en.md">English</a> · <strong>Español</strong></p>

![Espacio de trabajo de IAPro Nexus](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus es una estación de trabajo local para trabajar con coding agents. Organiza proyectos, agentes persistentes, sesiones, terminales supervisados, proveedores, worktrees y uso/cuota alrededor de un Core común. El camino más sencillo es: abrir un proyecto, crear una AI Session, elegir un proveedor y trabajar en la terminal. Composer, Flow, Mission y Maestro son capas progresivas, no requisitos para el trabajo directo.

## Empieza en minutos

El candidato actual `1.0.0-beta.2` integra Linux amd64, Windows amd64 y
macOS arm64 en el source, pero la certificación sigue ligada a evidencia
exact-SHA por plataforma.

Actualmente no hay una GitHub Release pública ni en el repositorio fuente ni en
`IAPro-Community/nexus-releases`. No construyas URLs de instalación usando la
tag histórica `v1.0.0-beta.1`.

Para evaluación de ingeniería, compila el source. La instalación binaria
firmada será el camino recomendado sólo después de que exista una release
publicada.

### Desarrolladores (compilar desde el código)

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
./nexus web
```

Después sigue **+ New → AI Session → proveedor → terminal**. No necesitas abrir Composer, Flow o Mission para trabajar directamente.

## Web, Desktop y CLI

| Superficie | Mejor para |
| --- | --- |
| **Nexus Web** | Control, observación y administración local desde el navegador. |
| **Nexus Desktop** | Shell nativo Wails cuando existan evidencias de build y runtime para la plataforma. |
| **Nexus CLI** | Scripts, diagnóstico, operación headless y flujos de terminal. |

Consulta la [matriz de plataformas](docs/operations/platform-support.md) antes de considerar soportada una capacidad nativa.

## Explora

- [Mapa de documentación](docs/README.md)
- [Getting started (índice)](docs/getting-started/README.md)
- [Instalación nativa](docs/getting-started/installation.md)
- [Paquetes nativos](docs/getting-started/packages.md)
- [Docker (compatibilidad)](docs/getting-started/docker-compat.md)
- [Tour visual](docs/product/visual-tour.md)
- [Flujo directo](docs/product/direct.md)
- [Composer](docs/product/composer.md)
- [Flow](docs/product/flow.md)
- [Desktop](docs/desktop/overview.md)
- [Arquitectura](docs/architecture/overview.md)
- [Troubleshooting](docs/operations/troubleshooting.md)

La [integración con Maestro](docs/product/maestro.md) es opcional para el camino Direct. Las afirmaciones sobre soporte, uso y continuidad dependen de evidencias; las capacidades planeadas o experimentales están marcadas.

## Comunidad

[Seguridad](SECURITY.md) · [Soporte](SUPPORT.md) · [Gobernanza](GOVERNANCE.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [Licencia Beta](LICENSE)
