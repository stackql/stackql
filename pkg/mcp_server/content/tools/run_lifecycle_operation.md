---
name: run_lifecycle_operation
---
Execute a stackql EXEC lifecycle operation (eg EXEC
aws.ec2.instances.start_instances ...). Real cloud side effects. Discover available operations and required params via list_methods and describe_method. Returns {messages, timestamp}. Gated by server mode.
