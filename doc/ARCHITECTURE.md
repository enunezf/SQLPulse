# Arquitectura del Sistema: SQLPulse

Se utiliza una **Arquitectura Hexagonal (Ports & Adapters)** para desacoplar la lógica de SQL Server de la interfaz de usuario (CLI).

## Capas
1. **/cmd:** Puntos de entrada de la aplicación. Define los comandos de Cobra.
2. **/internal/core:** - **Domain:** Modelos (Table, Index, QueryStat).
   - **Services:** Lógica de comparación de esquemas y análisis de rendimiento.
3. **/internal/adapters:**
   - **SQLServer:** Implementación de consultas a DMVs y extracción de DDL.
   - **Presenter:** Formateo de tablas y visualización de planes de ejecución.
4. **/internal/optimizer:** - Motor de reglas para sugerencias.
   - Interfaz para futura integración con APIs de IA.

## Flujo de Seguridad (Dry Run)
Ninguna sentencia `CREATE`, `ALTER` o `DROP` debe ejecutarse sin:
1. Generación de un script temporal.
2. Presentación del impacto al usuario.
3. Confirmación explícita mediante entrada de teclado `(y/n)`.