# Especificaciones Técnicas

## F1: Extractor de DDL
- Debe consultar `sys.objects` y `sys.sql_modules`.
- Capacidad de reconstruir el `CREATE TABLE` incluyendo Primary Keys y Clustered Indexes.

## F2: Comparador de Esquemas (Diff)
- Comparar `Source` vs `Target`.
- Detectar: Columnas faltantes, diferencias en tipos de datos, índices inexistentes.
- Generar salida tipo "diff" de Git.

## F3: Telemetría de Rendimiento
- **CPU/RAM:** Consultar `sys.dm_os_ring_buffers`.
- **Top Queries:** Basado en `sys.dm_exec_query_stats` ordenado por `total_worker_time`.

## F4: Planes de Ejecución
- Comando: `sqlpulse analyze --query "SELECT..."`
- Acción: Ejecutar `SET SHOWPLAN_XML ON`, capturar el XML y resaltar nodos con `EstimateIO` o `EstimateCPU` elevados.

## F5: Motor de Optimización
- El sistema debe sugerir índices basados en la vista `sys.dm_db_missing_index_details`.
- Las sugerencias de IA (v2) enviarán el esquema y el plan de ejecución como contexto a la API.