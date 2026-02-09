# SQLPulse CLI 🚀

**SQLPulse** es una herramienta de alto rendimiento escrita en Go para la auditoría, comparación y optimización de bases de datos SQL Server. 

### Características Principales
- **Análisis de Rendimiento:** Identificación de cuellos de botella mediante DMVs.
- **Gestión de Esquemas:** Extracción de DDL y comparación de bases de datos.
- **Optimización Segura:** Enfoque "Dry Run" para sugerencias de mejora con aprobación manual.
- **Extensibilidad:** Arquitectura preparada para integración con modelos de IA (Claude/OpenAI).

### Stack Tecnológico
- **Lenguaje:** Go 1.21+
- **CLI Framework:** Cobra
- **Database Driver:** go-mssqldb
- **TUI:** Bubbletea / Lipgloss