# Aliver
## Make your cluster to do the thing by only one server (with hot replacement)

## WARNING: In development

### How to start using
1) download binary for your platform from release section (if there is no one, coming soon)
2) write config file like config.yml in source code
```yml
app:
  cluster_id: 'cluster' // must be equal to all of the servers in cluster
  instance_id: 1 // must be unique in cluster
  ip: '127.0.0.1' // ip of the machine
  port: 6223 // must be equal to all of the servers in cluster
  mode: 'master' // [ follower ] - initial state for machine
  weight: 10 // probability of choosing to be the leader

  check_script: 'scripts/check.sh' // script to be executed to check whether the service dead or alive
  check_interval: '5s'
  check_retries: 5
  check_timeout: '10s'

  run_script: 'scripts/run.sh' // script to run the service (while being leader)
  run_timeout: '10s'

  stop_script: 'scripts/stop.sh' // script to stop the service (while being follower)
  stop_timeout: '10s'
```
3) set ALIVER_ENV environment variable to "CLUSTER"
4) set ALIVER_CONFIG_PATH environment variable to whether your config file is
5) start Aliver by executing the binary and fix the errors if occur
6) repeat steps 2-5 for all machines in cluster
7) enjoy