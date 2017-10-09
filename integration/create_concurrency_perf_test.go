package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/integration"
	"code.cloudfoundry.org/grootfs/integration/runner"
	"code.cloudfoundry.org/grootfs/store/locksmith"
	"code.cloudfoundry.org/lager"
	"gopkg.in/go-playground/pool.v3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Concurrent creations", func() {
	var workDir string

	BeforeEach(func() {
		err := Runner.RunningAsUser(0, 0).InitStore(runner.InitSpec{
			UIDMappings: []groot.IDMappingSpec{
				{HostID: GrootUID, NamespaceID: 0, Size: 1},
				{HostID: 100000, NamespaceID: 1, Size: 65000},
			},
			GIDMappings: []groot.IDMappingSpec{
				{HostID: GrootGID, NamespaceID: 0, Size: 1},
				{HostID: 100000, NamespaceID: 1, Size: 65000},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		workDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		Runner = Runner.SkipInitStore().WithLogLevel(lager.FATAL)
	})

	FIt("performance test", func() {
		baseImage := "assets/image.tar"
		for i := 0; i < 30; i++ {
			os.Chtimes(baseImage, time.Now(), time.Now())
			create(createCreateSpecWithNoDelete(fmt.Sprintf("cwnd-%d", i), baseImage))
			time.Sleep(time.Second)
		}

		for i := 0; i < 30; i++ {
			Expect(Runner.Delete(fmt.Sprintf("cwnd-%d", i))).To(Succeed())
		}

		lockFile, lock := obtainExclusiveLock()

		workerFunc := func(spec groot.CreateSpec) pool.WorkFunc {
			return func(wu pool.WorkUnit) (interface{}, error) {
				defer GinkgoRecover()
				fmt.Fprintf(os.Stderr, "---- > RUNNING SPEC! %#v\n", spec)
				stats := create(spec)
				return stats, nil
			}
		}

		createCache := func(wu pool.WorkUnit) (interface{}, error) {
			baseImage := "assets/image2.tar"
			os.Chtimes(baseImage, time.Now(), time.Now())
			create(createCreateSpecWithNoDelete("garbage-creator", baseImage))
			Expect(Runner.Delete("garbage-creator")).To(Succeed())
			return nil, nil
		}

		cwndPool := pool.NewLimited(10)
		cwdPool := pool.NewLimited(10)
		garbagePool := pool.NewLimited(1)

		cwndBatch := cwndPool.Batch()
		cwdBatch := cwdPool.Batch()
		garbageBatch := garbagePool.Batch()

		for i := 0; i < 5000; i++ {
			cwndBatch.Queue(workerFunc(createCreateSpecWithNoDelete(fmt.Sprintf("cwnd-parallel-%d", i), baseImage)))
			cwdBatch.Queue(workerFunc(createCreateSpecWithDelete(fmt.Sprintf("cwd-parallel-%d", i), baseImage)))
			garbageBatch.Queue(createCache)
		}

		// go func() {
		// 	defer GinkgoRecover()
		// 	time.Sleep(5 * time.Millisecond)
		// 	runner := Runner.WithLogLevel(lager.ERROR) // clone runner to avoid data-race on stdout
		// 	_, err := runner.Clean(0)
		// 	Expect(err).NotTo(HaveOccurred())
		// }()

		lock.Unlock(lockFile)

		garbageBatch.QueueComplete()
		cwdBatch.QueueComplete()
		cwndBatch.QueueComplete()

		cwdDataPoints := [][]float64{}
		for wu := range cwdBatch.Results() {
			stat := wu.Value().(CreateStats)
			cwdDataPoints = append(cwdDataPoints, []float64{float64(stat.FinishedAt.Unix()), float64(stat.CreationTime)})
		}
		sendData(cwdDataPoints, "cwd")

		cwndDataPoints := [][]float64{}
		for wu := range cwndBatch.Results() {
			stat := wu.Value().(CreateStats)
			cwndDataPoints = append(cwndDataPoints, []float64{float64(stat.FinishedAt.Unix()), float64(stat.CreationTime)})
		}
		sendData(cwndDataPoints, "cwnd")

		Expect(1).To(Equal(0))
	})

})

func createCreateSpec(id string, baseImageUrl string) groot.CreateSpec {
	return groot.CreateSpec{
		ID:           id,
		BaseImageURL: integration.String2URL(baseImageUrl),
		Mount:        mountByDefault(),
	}
}

func createCreateSpecWithNoDelete(id string, baseImageUrl string) groot.CreateSpec {
	return createCreateSpec(id, baseImageUrl)
}

func createCreateSpecWithDelete(id string, baseImageUrl string) groot.CreateSpec {
	spec := createCreateSpec(id, baseImageUrl)
	spec.CleanOnCreate = true
	return spec
}

func create(createSpec groot.CreateSpec) CreateStats {
	//time.Sleep(5 * time.Millisecond)
	runner := Runner.WithLogLevel(lager.ERROR) // clone runner to avoid data-race on stdout
	startTime := time.Now()
	_, err := runner.Create(createSpec)
	stat := CreateStats{
		ID:           createSpec.ID,
		CreationTime: int64(time.Since(startTime)),
		FinishedAt:   time.Now(),
	}
	Expect(err).NotTo(HaveOccurred())
	return stat
}

func obtainExclusiveLock() (*os.File, *locksmith.FileSystem) {
	lock := locksmith.NewExclusiveFileSystem(StorePath, nil)
	lockFile, err := lock.Lock(groot.GlobalLockKey)
	Expect(err).NotTo(HaveOccurred())
	return lockFile, lock
}

func releaseExclusiveLock(lockFile *os.File, lock *locksmith.FileSystem) {
	Expect(lock.Unlock(lockFile)).To(Succeed())
}

type CreateStats struct {
	ID           string
	CreationTime int64
	FinishedAt   time.Time
}

func sendData(datapoints [][]float64, tag string) {
	metricsEndpoint := "https://app.datadoghq.com/api/v1/series?api_key=" + os.Getenv("DATADOG_API_KEY")
	defer GinkgoRecover()

	series := map[string]interface{}{
		"metric": "grootfs-clean-tests",
		"type":   "gauge",
		"host":   "grootfs-ci",
		"points": datapoints,
		"tags":   []string{"clean-on-create", tag},
	}

	metric := map[string]interface{}{
		"series": []map[string]interface{}{series},
	}

	buf, _ := json.Marshal(metric)
	fmt.Fprintf(os.Stderr, "---> %s\n", string(buf))
	resp, err := http.Post(metricsEndpoint, "application/json", bytes.NewReader(buf))
	Expect(err).NotTo(HaveOccurred())
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(os.Stderr, "RESP BODY -> %s\n", string(respBody))
	Expect(resp.StatusCode).To(Equal(202))
}
