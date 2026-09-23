# IAPro Nexus

> **Estado de release:** Nexus v1 tiene como objetivo estable soportado **Linux amd64**.
> Windows y macOS permanecen en el source, pero están diferidos hasta completar su
> certificación nativa. La documentación pública registra `v1.0.0-beta.1` como la
> beta publicada.

**Una estación de trabajo local para operar coding agents.**

Proyectos, agentes persistentes, sesiones de terminal supervisadas, proveedores, worktrees y automatización en una experiencia integrada de Web, Desktop y CLI.

<p align="center"><a href="README.md">Português</a> · <a href="README.en.md">English</a> · <strong>Español</strong></p>

![Espacio de trabajo de IAPro Nexus](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus es una estación de trabajo local para trabajar con coding agents. Organiza proyectos, agentes persistentes, sesiones, terminales supervisados, proveedores, worktrees y uso/cuota alrededor de un Core común. El camino más sencillo es: abrir un proyecto, crear una AI Session, elegir un proveedor y trabajar en la terminal. Composer, Flow, Mission y Maestro son capas progresivas, no requisitos para el trabajo directo.

## Empieza en minutos

La matriz estable de Nexus v1 soporta **Linux amd64**. Windows y macOS no se publican
como targets soportados en esta versión.

Camino recomendado: instala una release precompilada y firmada.

```bash
curl -fsSL https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0-beta.1/install.sh -o install.sh
bash install.sh --version=v1.0.0-beta.1
nexus doctor
nexus web
```

El instalador exige manifiesto Ed25519 válido, SHA-256 coincidente y versión interna
exacta. Mientras la línea pública sea beta, usa la tag beta explícita.

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

El instalador se descarga del repositorio público de binarios. Para el payload exige un manifiesto Ed25519 válido, SHA-256 coincidente y que la versión embebida del binario sea exactamente la solicitada. Mientras la línea pública sea beta, use la tag beta explícita; `latest` representa la latest no-prerelease de GitHub.

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
