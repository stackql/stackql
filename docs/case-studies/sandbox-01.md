

Self join:
```sql
select 
  d1.name as n, 
  d1.id, 
  d2.id as d2_id 
from 
  google.compute.disks d1 
  inner join 
  google.compute.disks d2 
  on d1.id = d2.id 
where 
  d1.project = 'stackql-dev-01' 
  and d1.zone = 'australia-southeast1-b' 
  and d2.project = 'stackql-dev-01' 
  and d2.zone = 'australia-southeast1-b'
;
```