package ep

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Tests the scattering when there's just one node - the whole thing should
// be short-circuited to act as a pass-through
func TestExchangeInit_closeAllConnectionsUponError(t *testing.T) {
	port := ":5551"
	ln, err := net.Listen("tcp", port)
	require.NoError(t, err)
	dist := NewDistributer(port, ln)

	port2 := ":5552"

	ctx := context.WithValue(context.Background(), distributerKey, dist)
	ctx = context.WithValue(ctx, allNodesKey, []string{port, port2})
	ctx = context.WithValue(ctx, masterNodeKey, port)
	ctx = context.WithValue(ctx, thisNodeKey, port)

	exchange := Scatter().(*exchange)
	err = exchange.Init(ctx)

	require.Error(t, err)
	require.Equal(t, "dial tcp :5552: connect: connection refused", err.Error())
	require.Equal(t, 1, len(exchange.conns))
	require.IsType(t, &shortCircuit{}, exchange.conns[0])
	require.True(t, (exchange.conns[0]).(*shortCircuit).closed, "open connections leak")
	require.NoError(t, dist.Close())
}

// UID should be unique per generated exchange function
func TestExchange_unique(t *testing.T) {
	s1 := Scatter().(*exchange)
	s2 := Scatter().(*exchange)
	s3 := Gather().(*exchange)
	require.NotEqual(t, s1.UID, s2.UID)
	require.NotEqual(t, s2.UID, s3.UID)
}

func TestPartition_AddsMembersToHashRing(t *testing.T) {
	port := ":5551"
	ln, err := net.Listen("tcp", port)
	require.NoError(t, err)
	dist := NewDistributer(port, ln)

	port2 := ":5552"
	ln, err = net.Listen("tcp", port2)
	require.NoError(t, err)
	peer2 := NewDistributer(port2, ln)

	port3 := ":5553"
	ln, err = net.Listen("tcp", port3)
	require.NoError(t, err)
	peer3 := NewDistributer(port3, ln)

	defer func() {
		require.NoError(t, dist.Close())
		require.NoError(t, peer2.Close())
		require.NoError(t, peer3.Close())
	}()

	allNodes := []string{port, port2, port3}
	ctx := context.WithValue(context.Background(), distributerKey, dist)
	ctx = context.WithValue(ctx, allNodesKey, allNodes)
	ctx = context.WithValue(ctx, masterNodeKey, port)
	ctx = context.WithValue(ctx, thisNodeKey, port)

	partition := Partition().(*exchange)
	partition.Init(ctx)

	members := partition.hashRing.Members()
	require.ElementsMatchf(t, members, allNodes, "%s != %s", members, allNodes)
}

func TestExchange_EncodePartitionFailsWithoutDataset(t *testing.T) {
	port := ":5551"
	ln, err := net.Listen("tcp", port)
	require.NoError(t, err)
	dist := NewDistributer(port, ln)

	port2 := ":5552"
	ln, err = net.Listen("tcp", port2)
	require.NoError(t, err)
	peer2 := NewDistributer(port2, ln)

	port3 := ":5553"
	ln, err = net.Listen("tcp", port3)
	require.NoError(t, err)
	peer3 := NewDistributer(port3, ln)

	defer func() {
		require.NoError(t, dist.Close())
		require.NoError(t, peer2.Close())
		require.NoError(t, peer3.Close())
	}()

	ctx := context.WithValue(context.Background(), distributerKey, dist)
	ctx = context.WithValue(ctx, allNodesKey, []string{port, port2, port3})
	ctx = context.WithValue(ctx, masterNodeKey, port)
	ctx = context.WithValue(ctx, thisNodeKey, port)

	partition := Partition().(*exchange)
	partition.Init(ctx)

	require.Error(t, partition.EncodePartition([]int{42}))
}

func TestExchange_GetPartitionEncoderIsConsistent(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	maxPort := 7000
	minPort := 6000
	randomPort := rand.Intn(maxPort-minPort) + minPort
	port := fmt.Sprintf(":%d", randomPort)
	ln, err := net.Listen("tcp", port)
	require.NoError(t, err)
	dist := NewDistributer(port, ln)

	port2 := fmt.Sprintf(":%d", randomPort+1)
	ln, err = net.Listen("tcp", port2)
	require.NoError(t, err)
	peer2 := NewDistributer(port2, ln)

	port3 := fmt.Sprintf(":%d", randomPort+2)
	ln, err = net.Listen("tcp", port3)
	require.NoError(t, err)
	peer3 := NewDistributer(port3, ln)

	defer func() {
		require.NoError(t, dist.Close())
		require.NoError(t, peer2.Close())
		require.NoError(t, peer3.Close())
	}()

	ctx := context.WithValue(context.Background(), distributerKey, dist)
	ctx = context.WithValue(ctx, allNodesKey, []string{port, port2, port3})
	ctx = context.WithValue(ctx, masterNodeKey, port)
	ctx = context.WithValue(ctx, thisNodeKey, port)

	partition := Partition().(*exchange)
	partition.Init(ctx)

	hashKey := fmt.Sprintf("this-is-a-key-%d", rand.Intn(1e12))
	enc, err := partition.getPartitionEncoder(hashKey)
	require.NoError(t, err)

	// verify that getPartitionEncoder returns the same result over 100 calls
	for i := 0; i < 100; i++ {
		nextEncoder, err := partition.getPartitionEncoder(hashKey)
		require.NoError(t, err)
		require.Equal(t, enc, nextEncoder)
	}
}
