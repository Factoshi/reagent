# ChockAgent

## Run the agent

The agent must be run on a machine with a local instance of factomd running.

```bash
# Replace AGENT_NAME environment variable value with a uniquely identifiable name
docker pull luciaptech/chockagent
docker run -d \
    --network host \
    --name chockagent \
    -e AGENT_NAME="luciap-testnet" \
    luciaptech/chockagent

# Verify the agent succesfully connected to the coordinator
docker logs chockagent
```
