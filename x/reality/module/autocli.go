package reality

import (
    autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

    "contactical/x/reality/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
    return &autocliv1.ModuleOptions{
        Query: &autocliv1.ServiceCommandDescriptor{
            Service: types.Query_serviceDesc.ServiceName,
            RpcCommandOptions: []*autocliv1.RpcCommandOptions{
                {
                    RpcMethod: "Params",
                    Use:       "params",
                    Short:     "Shows the parameters of the module",
                },
                {
                    RpcMethod: "ListClaim",
                    Use:       "list-claim",
                    Short:     "List all claim",
                },
                {
                    RpcMethod:      "GetClaim",
                    Use:            "get-claim [id]",
                    Short:          "Gets a claim by id",
                    Alias:          []string{"show-claim"},
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
                },
                // this line is used by ignite scaffolding # autocli/query
            },
        },
        Tx: &autocliv1.ServiceCommandDescriptor{
            Service:              types.Msg_serviceDesc.ServiceName,
            EnhanceCustomCommand: true, // only required if you want to use the custom command
            RpcCommandOptions: []*autocliv1.RpcCommandOptions{
                {
                    RpcMethod: "UpdateParams",
                    Skip:      true, // skipped because authority gated
                },
                {
                    RpcMethod:      "CreateClaim",
                    Use:            "create-claim [sensor-hash] [gnss-hash] [anchor-signature]",
                    Short:          "Send a create-claim tx",
                    PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "sensor_hash"}, {ProtoField: "gnss_hash"}, {ProtoField: "anchor_signature"}},
                },
                {
                    RpcMethod: "Swap",
                    Use:       "swap --amount-in [amount] --target-denom [denom]",
                    Short:     "Swap tokens using DEX",
                },
                // this line is used by ignite scaffolding # autocli/tx
            },
        },
    }
}
