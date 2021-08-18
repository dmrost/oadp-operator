package e2e

import (
	"context"
	"fmt"
	"log"
	"time"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func hasCorrectNumResticPods(namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		client, err := setUpClient()
		if err != nil {
			return false, err
		}
		resticOptions := metav1.ListOptions{
			FieldSelector: "metadata.name=restic",
		}
		resticDaemeonSet, err := client.AppsV1().DaemonSets(namespace).List(context.TODO(), resticOptions)
		if err != nil {
			return false, err
		}
		var numScheduled int32
		var numDesired int32

		for _, daemonSetInfo := range (*resticDaemeonSet).Items {
			numScheduled = daemonSetInfo.Status.CurrentNumberScheduled
			numDesired = daemonSetInfo.Status.DesiredNumberScheduled
		}
		// check correct num of Restic pods are initialized
		if numScheduled != 0 && numDesired != 0 {
			if numScheduled == numDesired {
				return true, nil
			}
		}
		return false, err
	}
}

func waitForDesiredResticPods(namespace string) error {
	return wait.PollImmediate(time.Second*5, time.Minute*2, hasCorrectNumResticPods(namespace))
}

func areResticPodsRunning(namespace string) wait.ConditionFunc {
	log.Printf("Checking for correct number of running Restic pods...")
	return func() (bool, error) {
		err := waitForDesiredResticPods(namespace)
		if err != nil {
			return false, err
		}
		client, err := setUpClient()
		if err != nil {
			return false, err
		}
		resticPodOptions := metav1.ListOptions{
			LabelSelector: "component=restic",
		}
		// get pods in the oadp-operator-e2e namespace with label selector
		podList, err := client.CoreV1().Pods(namespace).List(context.TODO(), resticPodOptions)
		if err != nil {
			return false, nil
		}
		// loop until pod status is 'Running' or timeout
		for _, podInfo := range (*podList).Items {
			if podInfo.Status.Phase != "Running" {
				return false, err
			} else {
				return true, nil
			}
		}
		return false, err
	}
}

func doesDaemonSetExists(namespace string, resticName string) wait.ConditionFunc {
	log.Printf("Checking if restic daemonset exists...")
	return func() (bool, error) {
		clientset, err := setUpClient()
		if err != nil {
			return false, err
		}
		_, err = clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), resticName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

// keep for now
func isResticDaemonsetDeleted(namespace string, instanceName string, resticName string) wait.ConditionFunc {
	log.Printf("Checking if Restic daemonset has been deleted...")
	return func() (bool, error) {
		client, err := setUpClient()
		if err != nil {
			return false, nil
		}
		// Check for daemonSet
		_, err = client.AppsV1().DaemonSets(namespace).Get(context.Background(), resticName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Restic daemonSet has been deleted")
			return true, nil
		}
		return false, err
	}
}

func (v *veleroCustomResource) disableRestic(namespace string, instanceName string) error {
	err := v.SetClient()
	if err != nil {
		return err
	}
	vel := oadpv1alpha1.Velero{}
	err = v.Client.Get(context.Background(), client.ObjectKey{
		Namespace: v.Namespace,
		Name:      v.Name,
	}, &vel)
	if err != nil {
		return err
	}

	// update spec 'enable_restic' to be false
	enable := vel.Spec.EnableRestic
	enable = pointer.Bool(false)
	err = v.Client.Update(context.Background(), &vel, enable)
	if err != nil {
		return err
	}
	fmt.Println("spec 'enable_restic' has been updated to false")
	return nil
}

// func enableResticNodeSelector(namespace string, s3Bucket string, credSecretRef string, instanceName string) error {
// 	veleroClient, err := setUpDynamicVeleroClient(namespace)
// 	if err != nil {
// 		return nil
// 	}
// 	// get Velero as unstructured type
// 	veleroResource := getResticVeleroConfig(namespace, s3Bucket, credSecretRef, instanceName)
// 	_, err = veleroClient.Create(context.Background(), veleroResource, metav1.CreateOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("spec 'restic_node_selector' has been updated")
// 	return nil
// }

func resticDaemonSetHasNodeSelector(namespace string, s3Bucket string, credSecretRef string, instanceName string, resticName string) wait.ConditionFunc {
	log.Printf("Waiting for Restic daemonset to have a nodeSelector...")
	return func() (bool, error) {
		client, err := setUpClient()
		if err != nil {
			return false, nil
		}
		ds, err := client.AppsV1().DaemonSets(namespace).Get(context.TODO(), resticName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// verify daemonset has nodeSelector "foo": "bar"
		selector := ds.Spec.Template.Spec.NodeSelector["foo"]

		if selector == "bar" {
			return true, nil
		}
		return false, err
	}
}
