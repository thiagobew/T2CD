rm -rf nodes
mkdir nodes
touch nodes/.gitkeep

# Create an array to store the PIDs
pids=()

# Run the programs in a loop
for ((i=0; i < $NODES; i++)); do
  haddr=$(($START_SERVER_PORT+i*100))
  raddr=$(($START_RAFT_PORT+i))

  if [ $i -ne 0 ]; then
    ./bin/main -haddr "localhost:$haddr" -raddr "localhost:$raddr" -id node$i -join "$LEADER_IP:$START_SERVER_PORT" ./nodes/node$i &
  else
    ./bin/main -haddr "localhost:$haddr" -raddr "localhost:$raddr" -id node$i ./nodes/node$i &
    sleep 3 # wait for the first node to start
  fi

  # Store the PID of the program
  pids+=($!)
done

# Wait for all programs to finish
wait

# Kill the programs when the script ends
for pid in "${pids[@]}"; do
  kill $pid
done