
## Dashboard software particulars

- Dashboard software can interrogate views. 
- Tables and views need to be defined *a priori* for queries to work.  
  This implies that some AOT pre-charging of tables is required.  Here are some options:
    - ~~(a) Dedicate some namespace to such queries.  For any query against the namespace, front load table and view creation and then proceed.~~ **This, on its own, will NOT work for creating dashboards because the dashboard software will be unaware of the requisite table / view objects.**
    - (b) Pre-charge all tables and views upon provider download.  Superficially, seems damned expensive.
    - (c) **Hack**: Run ad hoc queries (or batches) to pre-charge from within dashboard (or `stackql`) as an offline, admin activity.
- Doing either (b) or (c) will require `stackql` to natively support subqueries.
- Could consider some `precharge <table_name_list> ;` grammar to facilitate.


## Data flow

1. Admin (or automated) action to pre-charge tables.
1. Admin user enters `stackql` query to pre-charge tables.
2. `stackql` executes query and returns results.



