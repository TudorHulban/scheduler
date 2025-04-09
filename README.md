# scheduler

Scheduler:  
Domain elements:

1. resource type  
with resource schedule which is a list of tasks per time intervals
each resource may have a usage cost per hour

2. task  
with needed resources and quantities and estimated duration

3. location  
which is a sum of tasks and resources for a physical business location

4. scheduler  
should be able:  

a. run or schedule a task  
b. calculate cost of task run  
c. suggest to move the task to other location if:  

- task cannot run within time interval in initial location
- task is cheaper to run in other location for same time interval
