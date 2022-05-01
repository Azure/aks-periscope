package test

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// requiredImages is the complete list of Docker images specified in containers
// when a test run is executed.
var requiredImages = []string{
	"docker.io/kindest/kindnetd:v20211122-a2c10462",
	"docker.io/rancher/local-path-provisioner:v0.0.14",
	"k8s.gcr.io/coredns/coredns:v1.8.6",
	"k8s.gcr.io/etcd:3.5.1-0",
	"k8s.gcr.io/kube-apiserver:v1.23.5",
	"k8s.gcr.io/kube-controller-manager:v1.23.5",
	"k8s.gcr.io/kube-proxy:v1.23.5",
	"k8s.gcr.io/kube-scheduler:v1.23.5",
	"k8s.gcr.io/metrics-server/metrics-server:v0.6.1",
	"nginx:1.16.0",
}

// use a map to emulate a distinct set with efficient lookup
// (populated in the init() method)
var requiredImageSet map[string]bool

// PullAndLoadDockerImages ensures all images required by all tests are pre-loaded on to the Kind cluster
// before running any tests. If this is *not* done, the images will not be pulled from their respective
// registries on every test run, and not cached on the host (because they are pulled from within the Docker
// containers comprising the Kind cluster, not the host itself).
func PullAndLoadDockerImages(client *dockerclient.Client, commandRunner *ToolsCommandRunner) error {
	images, err := client.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("error listing Docker images: %w", err)
	}

	availableImageSet := make(map[string]bool)
	for _, image := range images {
		for _, tag := range image.RepoTags {
			availableImageSet[tag] = true
		}
	}

	imagesToPull := []string{}
	for _, image := range requiredImages {
		if _, ok := availableImageSet[image]; !ok {
			imagesToPull = append(imagesToPull, image)
		}
	}

	err = pullDockerImages(client, imagesToPull)
	if err != nil {
		return fmt.Errorf("error pulling Docker images: %w", err)
	}

	loadDockerImagesCommand := GetLoadDockerImagesCommand(requiredImages)
	_, err = commandRunner.Run(loadDockerImagesCommand)
	if err != nil {
		return fmt.Errorf("error loading Docker images into cluster: %w", err)
	}

	return nil
}

func pullDockerImages(client *dockerclient.Client, imagesToPull []string) error {
	// Pull the images in parallel.
	// Use channels to return the first error, or return when completed.
	pullErrorsChan := make(chan error)
	wgDoneChan := make(chan bool)

	wg := new(sync.WaitGroup)
	wg.Add(len(imagesToPull))

	for _, image := range imagesToPull {
		go func(image string) {
			defer wg.Done()
			pullOutput, err := client.ImagePull(context.Background(), image, dockertypes.ImagePullOptions{})
			if err != nil {
				pullErrorsChan <- fmt.Errorf("error pulling image %s: %w", image, err)
			}
			defer pullOutput.Close()
			_, err = io.Copy(os.Stdout, pullOutput)
			if err != nil {
				pullErrorsChan <- fmt.Errorf("error copying image pull output to stdout: %w", err)
			}
		}(image)
	}

	go func() {
		wg.Wait()
		close(wgDoneChan)
	}()

	select {
	case <-wgDoneChan:
		return nil
	case err := <-pullErrorsChan:
		close(pullErrorsChan)
		return err
	}
}

// CheckDockerImages checks our list of required images is up-to-date based on pods currently running in the test cluster.
// If any images are superfluous or missing it will return an error specifying the image tags that need to be added or removed.
// It also verifies the pull policies to ensure that no unnecessary downloading of images occurs during test runs.
func CheckDockerImages(clientset *kubernetes.Clientset) error {
	podList, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pods in all namespaces: %w", err)
	}

	pullAlwaysContainers := []string{}
	actualImageSet := make(map[string]bool)
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			actualImageSet[container.Image] = true
			if container.ImagePullPolicy == corev1.PullAlways {
				pullAlwaysContainers = append(pullAlwaysContainers, fmt.Sprintf("%s/%s", pod.Name, container.Name))
			}
		}
	}

	// Avoid any 'Always' pull policies in tests
	if len(pullAlwaysContainers) > 0 {
		return fmt.Errorf("pull policy 'always' not permitted for tests, found in:\n%s", strings.Join(pullAlwaysContainers, "\n"))
	}

	// Check missing requirements
	missingRequirements := []string{}
	for image := range actualImageSet {
		if _, ok := requiredImageSet[image]; !ok {
			missingRequirements = append(missingRequirements, image)
		}
	}
	if len(missingRequirements) > 0 {
		return fmt.Errorf("missing images in requiredImages slice:\n%s", strings.Join(missingRequirements, "\n"))
	}

	// Check superfluous requirements
	superfluousRequirements := []string{}
	for image := range requiredImageSet {
		if _, ok := actualImageSet[image]; !ok {
			superfluousRequirements = append(superfluousRequirements, image)
		}
	}
	if len(superfluousRequirements) > 0 {
		return fmt.Errorf("superfluous images in requiredImages slice:\n%s", strings.Join(superfluousRequirements, "\n"))
	}

	return nil
}

func init() {
	requiredImageSet = make(map[string]bool)
	for _, image := range requiredImages {
		requiredImageSet[image] = true
	}
}
