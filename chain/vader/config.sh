#!/bin/bash

#rm -rf ~/.vader
vader config chain-id band-guanyu-mainnet
#vader config node http://localhost:26657
vader config node tcp://75271c4294f6408c0a90783b18d6524b426b555c@65.21.145.202:26657
vader keys add anakin
vader config requester $(bandcli keys show anakin -a)
vader config broadcast-timeout "1m"
vader config rpc-poll-interval "1s"
vader config max-try 5

vader config ask-count 3
vader config min-count 5
vader config oracle-script-id 37
vader config symbols "BTC,ETH"
