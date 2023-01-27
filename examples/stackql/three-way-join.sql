select 
  d1.name as n, 
  d1.id, 
  n1.description, 
  s1.description as s1_description 
from 
  google.compute.disks d1 
  inner join google.compute.networks n1 
  on 
    d1.name = n1.name 
  inner join 
  google.compute.subnetworks s1 
  on 
    d1.name = s1.name  
where 
  d1.project = 'testing-project' and 
  d1.zone = 'australia-southeast1-b' and 
  n1.project = 'testing-project' 
  and s1.project = 'testing-project' 
  and s1.region = 'australia-southeast1'
;