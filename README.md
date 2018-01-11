# Advanced AdWords Query Language (AWQL)

AWQL is a SQL-like language for performing queries against most common AdWords API services.

This repository exports in one library all the features added by the [AWQL Command-Line Tool](https://github.com/rvflash/awql "The AWQL Command-Line Tool") to the AWQL grammar : 

* Adds the SQL clauses to SELECT statement: LIMIT, GROUP BY and ORDER BY.
* Also adds the aggregate functions: AVG, COUNT, MAX, MIN, SUM and DISTINCT keyword.
* Caching data.


See these repositories as source of work and docs:
* https://github.com/rvflash/awql-db
* https://github.com/rvflash/awql-driver
* https://github.com/rvflash/awql-parser
* https://github.com/rvflash/csv-cache