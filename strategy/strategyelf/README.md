# ndn-dpdk/strategy/strategyelf

This package embeds compiled strategy BPF program.
`go-bindata` tool compiles the ELF objects into `bindata.go`.
Go code can access them with `Load` function.
