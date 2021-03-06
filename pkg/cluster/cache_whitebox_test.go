package cluster

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var getFedClusterFuncs = []func(name string) (*FedCluster, bool){
	clusterCache.getFedCluster, GetFedCluster}

func TestAddCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, ready)

	// when
	clusterCache.addFedCluster(fedCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func TestGetCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, ready)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, ready))

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.True(t, ok)
		assert.Equal(t, fedCluster, returnedFedCluster)
	}
}

func TestHostCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	host := newTestFedCluster("host-cluster", Host, ready)
	clusterCache.addFedCluster(host)

	// when
	returnedFedCluster, ok := HostCluster()

	// then
	assert.True(t, ok)
	assert.Equal(t, host, returnedFedCluster)
}

func TestMemberClusters(t *testing.T) {
	// given
	defer resetClusterCache()
	member1 := newTestFedCluster("member-cluster-1", Member, ready)
	clusterCache.addFedCluster(member1)
	member2 := newTestFedCluster("member-cluster-2", Member, ready)
	clusterCache.addFedCluster(member2)

	// when
	returnedFedClusters := MemberClusters()

	// then
	require.Len(t, returnedFedClusters, 2)
	assert.Contains(t, returnedFedClusters, member1)
	assert.Contains(t, returnedFedClusters, member2)
}

func TestGetClusterWhenIsEmpty(t *testing.T) {
	// given
	resetClusterCache()

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedFedCluster)
	}
}

func TestGetClustersByType(t *testing.T) {

	t.Run("get clusters by type", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// empty cache

			//when
			clusters := clusterCache.getFedClustersByType(Member)

			//then
			assert.Empty(t, clusters)

			//when
			clusters = clusterCache.getFedClustersByType(Host)

			//then
			assert.Empty(t, clusters)
		})

		t.Run("found", func(t *testing.T) {
			defer resetClusterCache()
			// given
			// Two members, one host
			member1 := newTestFedCluster("cluster-1", Member, ready)
			clusterCache.addFedCluster(member1)
			member2 := newTestFedCluster("cluster-2", Member, ready)
			clusterCache.addFedCluster(member2)
			host := newTestFedCluster("cluster-3", Host, ready)
			clusterCache.addFedCluster(host)

			//when
			clusters := clusterCache.getFedClustersByType(Member)

			//then
			assert.Len(t, clusters, 2)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)

			//when
			clusters = clusterCache.getFedClustersByType(Host)

			//then
			assert.Len(t, clusters, 1)
			assert.Contains(t, clusters, host)
		})
	})

	t.Run("get member clusters", func(t *testing.T) {
		// noise
		host := newTestFedCluster("cluster-host", Host, ready)
		clusterCache.addFedCluster(host)

		t.Run("not found", func(t *testing.T) {
			// given
			defer resetClusterCache()
			// no members

			//when
			clusters := GetMemberClusters()

			//then
			assert.Empty(t, clusters)
		})

		t.Run("all clusters", func(t *testing.T) {
			// given
			defer resetClusterCache()
			member1 := newTestFedCluster("cluster-1", Member, ready)
			clusterCache.addFedCluster(member1)
			member2 := newTestFedCluster("cluster-2", Member, ready)
			clusterCache.addFedCluster(member2)

			//when
			clusters := GetMemberClusters()

			//then
			assert.Len(t, clusters, 2)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
		})

		t.Run("found after refreshing the cache", func(t *testing.T) {
			// given
			defer resetClusterCache()
			member := newTestFedCluster("member", Member, ready)
			called := false
			clusterCache.refreshCache = func() {
				called = true
				clusterCache.addFedCluster(member)
			}

			//when
			clusters := GetMemberClusters()

			//then
			assert.Len(t, clusters, 1)
			assert.Contains(t, clusters, member)
			assert.True(t, called)
		})

	})

	t.Run("get member clusters filtered by readiness and capacity", func(t *testing.T) {
		defer resetClusterCache()

		// noise
		host := newTestFedCluster("cluster-host", Host, ready)
		clusterCache.addFedCluster(host)
		member1 := newTestFedCluster("cluster-1", Member, ready)
		clusterCache.addFedCluster(member1)
		member2 := newTestFedCluster("cluster-2", Member, ready)
		clusterCache.addFedCluster(member2)
		member3 := newTestFedCluster("cluster-3", Member, notReady)
		clusterCache.addFedCluster(member3)
		member4 := newTestFedCluster("cluster-4", Member, ready, capacityExhausted)
		clusterCache.addFedCluster(member4)

		t.Run("get only ready member clusters", func(t *testing.T) {
			//when
			clusters := GetMemberClusters(Ready)

			//then
			assert.Len(t, clusters, 3)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
			assert.Contains(t, clusters, member4)
		})

		t.Run("get only member clusters with free capacity", func(t *testing.T) {
			//when
			clusters := GetMemberClusters(CapacityNotExhausted)

			//then
			assert.Len(t, clusters, 3)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
			assert.Contains(t, clusters, member3)
		})

		t.Run("get only ready member clusters that have free capacity", func(t *testing.T) {
			//when
			clusters := GetMemberClusters(Ready, CapacityNotExhausted)

			//then
			assert.Len(t, clusters, 2)
			assert.Contains(t, clusters, member1)
			assert.Contains(t, clusters, member2)
		})
	})

	t.Run("get host cluster", func(t *testing.T) {

		// noise
		member1 := newTestFedCluster("cluster-member-1", Member, ready)
		clusterCache.addFedCluster(member1)

		t.Run("not found", func(t *testing.T) {
			// given
			defer resetClusterCache()
			// no host

			//when
			_, ok := GetHostCluster()

			//then
			assert.False(t, ok)
		})

		t.Run("found", func(t *testing.T) {
			// given
			defer resetClusterCache()
			host := newTestFedCluster("cluster-host", Host, ready)
			clusterCache.addFedCluster(host)

			//when
			cluster, ok := GetHostCluster()

			//then
			assert.True(t, ok)
			assert.Equal(t, host, cluster)
		})

		t.Run("found after refreshing the cache", func(t *testing.T) {
			// given
			defer resetClusterCache()
			host := newTestFedCluster("cluster-host", Host, ready)
			called := false
			clusterCache.refreshCache = func() {
				called = true
				clusterCache.addFedCluster(host)
			}

			//when
			cluster, ok := GetHostCluster()

			//then
			assert.True(t, ok)
			assert.Equal(t, host, cluster)
			assert.True(t, called)
		})
	})
}

func TestGetClusterUsingDifferentKey(t *testing.T) {
	// given
	defer resetClusterCache()
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, ready))

	for _, getFedCluster := range getFedClusterFuncs {

		// when
		returnedFedCluster, ok := getFedCluster("testCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, returnedFedCluster)
	}
}

func TestUpdateCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	trueCluster := newTestFedCluster("testCluster", Member, ready)
	falseCluster := newTestFedCluster("testCluster", Member, notReady)
	clusterCache.addFedCluster(trueCluster)

	// when
	clusterCache.addFedCluster(falseCluster)

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, falseCluster, clusterCache.clusters["testCluster"])
}

func TestDeleteCluster(t *testing.T) {
	// given
	defer resetClusterCache()
	fedCluster := newTestFedCluster("testCluster", Member, ready)
	clusterCache.addFedCluster(fedCluster)
	clusterCache.addFedCluster(newTestFedCluster("cluster", Member, ready))
	assert.Len(t, clusterCache.clusters, 2)

	// when
	clusterCache.deleteFedCluster("cluster")

	// then
	assert.Len(t, clusterCache.clusters, 1)
	assert.Equal(t, fedCluster, clusterCache.clusters["testCluster"])
}

func TestRefreshCache(t *testing.T) {
	// given
	defer resetClusterCache()
	testCluster := newTestFedCluster("testCluster", Member, ready)
	newCluster := newTestFedCluster("newCluster", Member, ready)
	clusterCache.addFedCluster(testCluster)
	clusterCache.refreshCache = func() {
		clusterCache.addFedCluster(newCluster)
	}

	t.Run("refresh and get existing cluster", func(t *testing.T) {
		// when
		returnedNewCluster, ok := clusterCache.getFedCluster("newCluster")

		// then
		assert.True(t, ok)
		assert.Equal(t, newCluster, returnedNewCluster)

		returnedTestCluster, ok := clusterCache.getFedCluster("testCluster")
		assert.True(t, ok)
		assert.Equal(t, testCluster, returnedTestCluster)
	})

	t.Run("refresh and get non-existing cluster", func(t *testing.T) {
		// when
		cluster, ok := clusterCache.getFedCluster("anotherCluster")

		// then
		assert.False(t, ok)
		assert.Nil(t, cluster)
	})
}

func TestMultipleActionsInParallel(t *testing.T) {
	// given
	defer resetClusterCache()
	var latch sync.WaitGroup
	latch.Add(1)
	var waitForFinished sync.WaitGroup

	memberCluster := newTestFedCluster("memberCluster", Member, ready)
	hostCluster := newTestFedCluster("hostCluster", Host, ready)
	clusterCache.refreshCache = func() {
		clusterCache.addFedCluster(memberCluster)
		clusterCache.addFedCluster(hostCluster)
	}

	for _, clusterToTest := range []*FedCluster{memberCluster, hostCluster} {
		for i := 0; i < 1000; i++ {
			waitForFinished.Add(4)
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				clusterCache.addFedCluster(clusterToTest)
			}()
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				cluster, ok := clusterCache.getFedCluster(clusterToTest.Name)
				if ok {
					assert.Equal(t, clusterToTest, cluster)
				} else {
					assert.Nil(t, cluster)
				}
			}()
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				clusters := clusterCache.getFedClustersByType(clusterToTest.Type)
				if len(clusters) == 1 {
					assert.Equal(t, clusterToTest, clusters[0])
				} else {
					assert.Empty(t, clusters)
				}
			}()
			go func() {
				defer waitForFinished.Done()
				latch.Wait()
				clusterCache.deleteFedCluster(clusterToTest.Name)
			}()
		}
	}

	// when
	latch.Done()

	// then
	waitForFinished.Wait()

	member, ok := clusterCache.getFedCluster("memberCluster")
	assert.True(t, ok)
	assert.Equal(t, memberCluster, member)

	host, ok := clusterCache.getFedCluster("hostCluster")
	assert.True(t, ok)
	assert.Equal(t, hostCluster, host)
}

// clusterOption an option to configure the cluster to use in the tests
type clusterOption func(*FedCluster)

// Ready an option to state the cluster as "ready"
var ready clusterOption = func(c *FedCluster) {
	c.ClusterStatus.Conditions = append(c.ClusterStatus.Conditions, v1beta1.ClusterCondition{
		Type:   common.ClusterReady,
		Status: v1.ConditionTrue,
	})
}

// clusterNotReady an option to state the cluster as "not ready"
var notReady clusterOption = func(c *FedCluster) {
	c.ClusterStatus.Conditions = append(c.ClusterStatus.Conditions, v1beta1.ClusterCondition{
		Type:   common.ClusterReady,
		Status: v1.ConditionFalse,
	})
}

// capacityExhausted an option to state that the cluster capacity has exhausted
var capacityExhausted clusterOption = func(c *FedCluster) {
	c.CapacityExhausted = true
}

func newTestFedCluster(name string, clusterType Type, options ...clusterOption) *FedCluster {
	cl := fake.NewFakeClient()
	fedCluster := &FedCluster{
		Name:              name,
		Client:            cl,
		OperatorNamespace: name + "Namespace",
		Type:              clusterType,
		CapacityExhausted: false,
		ClusterStatus:     &v1beta1.KubeFedClusterStatus{},
	}
	for _, configure := range options {
		configure(fedCluster)
	}
	return fedCluster
}

func resetClusterCache() {
	clusterCache = kubeFedClusterClients{clusters: map[string]*FedCluster{}}
}
