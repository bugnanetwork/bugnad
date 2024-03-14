package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bugnanetwork/bugnad/infrastructure/network/netadapter/server/grpcserver/protowire"
)

var commandTypes = []reflect.Type{
	reflect.TypeOf(protowire.BugnadMessage_AddPeerRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetConnectedPeerInfoRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetPeerAddressesRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetCurrentNetworkRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetInfoRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_GetBlockRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetBlocksRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetHeadersRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetBlockCountRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetBlockDagInfoRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetSelectedTipHashRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetVirtualSelectedParentBlueScoreRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetVirtualSelectedParentChainFromBlockRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_ResolveFinalityConflictRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_EstimateNetworkHashesPerSecondRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_GetBlockTemplateRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_SubmitBlockRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_GetMempoolEntryRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetMempoolEntriesRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetMempoolEntriesByAddressesRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_SubmitTransactionRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_GetUtxosByAddressesRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetBalanceByAddressRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_GetCoinSupplyRequest{}),

	reflect.TypeOf(protowire.BugnadMessage_BanRequest{}),
	reflect.TypeOf(protowire.BugnadMessage_UnbanRequest{}),
}

type commandDescription struct {
	name       string
	parameters []*parameterDescription
	typeof     reflect.Type
}

type parameterDescription struct {
	name   string
	typeof reflect.Type
}

func commandDescriptions() []*commandDescription {
	commandDescriptions := make([]*commandDescription, len(commandTypes))

	for i, commandTypeWrapped := range commandTypes {
		commandType := unwrapCommandType(commandTypeWrapped)

		name := strings.TrimSuffix(commandType.Name(), "RequestMessage")
		numFields := commandType.NumField()

		var parameters []*parameterDescription
		for i := 0; i < numFields; i++ {
			field := commandType.Field(i)

			if !isFieldExported(field) {
				continue
			}

			parameters = append(parameters, &parameterDescription{
				name:   field.Name,
				typeof: field.Type,
			})
		}
		commandDescriptions[i] = &commandDescription{
			name:       name,
			parameters: parameters,
			typeof:     commandTypeWrapped,
		}
	}

	return commandDescriptions
}

func (cd *commandDescription) help() string {
	sb := &strings.Builder{}
	sb.WriteString(cd.name)
	for _, parameter := range cd.parameters {
		_, _ = fmt.Fprintf(sb, " [%s]", parameter.name)
	}
	return sb.String()
}
