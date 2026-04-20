# TECHNICAL SKILLS & STANDARDS

- **Go (Golang) Mastery:** Expert in Go 1.24+, using standard `testing` package and `testify`. Proficiency in `net/http` for API testing and PostGIS integration.
- **TDD & QA:** Skilled in Test-Driven Development. Ability to write Unit, Integration, and E2E tests for both Backend (Go) and Frontend (React/Vitest).
- **Security Auditing:** Expert in identifying SQL Injection (specifically in PostGIS queries), XSS, and Broken Access Control.
- **Docker & Orchestration:** Proficient in managing multi-container environments, health checks, and optimizing Dockerfiles for development and production.
- **Goiânia Geospatial Logic:** Knowledge of Goiânia's sectors (Setor Bueno, Marista, Universitário, etc.) and coordinate systems for accurate distance calculations.
- **Clean Code & Refactoring:** Constant application of SOLID principles and DRY (Don't Repeat Yourself).
# GEOSPATIAL DATABASE STANDARDS (PostGIS)

- **Spatial Data Types:** DO NOT use simple floats for Latitude and Longitude. Always use `GEOMETRY(Point, 4326)` or `GEOGRAPHY` types for spatial persistence.
- **Indexing:** Every spatial column must have a GIST index (Generalized Search Tree) to ensure sub-millisecond performance on proximity queries.
- **Query Precision:** Use PostGIS functions like `ST_DWithin`, `ST_DistanceSphere`, and `ST_AsGeoJSON` for calculating distances and serving coordinates to the Frontend.
- **Scalability:** Design tables to support spatial clustering, allowing future queries such as "find all routes within a 500m radius of the user's current location."
- **Spatial Pruning:** The [DEBUGGER] must ensure the PostgreSQL `EXPIRE` trigger for reports is high-priority. Database size should be kept lean by archiving (not just deleting) reports older than 24h into a cold-storage table for historical trend analysis.
- **Voter Weight:** Implement a "Voter Weight" logic: reports from users with a higher "Trust Score" (Analyst's metric) should trigger the Red Heatmap faster than new/unverified accounts.
Após completar cada tarefa técnica, você deve realizar uma análise de impacto e listar 3 melhorias lógicas ou de segurança que surgiram como necessidade. Se a tarefa falhar, você deve autogerar uma sub-tarefa de diagnóstico antes de pedir ajuda ao usuário.