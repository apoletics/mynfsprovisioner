/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
//	"errors"
	"flag"
	"os"
//	"path"
	"syscall"

//	"bytes"
//	"crypto/sha256"
//	"fmt"
//	"io"
//	"io/ioutil"
//	"log"
//	"strings"

	"nfs"
	"rpc"
	"util"
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/controller"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	provisionerName = "example.com/mynfsprovisioner"
)

type nfsPathProvisioner struct {
	nfsHost string
	nfsBasepath string
	nfsMode string
	nfsOwnerid string
}

// NewHostPathProvisioner creates a new hostpath provisioner
func NewNfsPathProvisioner() controller.Provisioner {
        nfsHost := os.Getenv("NFS_HOST")
        if nfsHost == "" {
                glog.Fatal("env variable NFS_HOST must be set so that this provisioner can identify itself")
        }

        nfsBasepath := os.Getenv("NFS_BASE_PATH")
        if nfsBasepath == "" {
                glog.Fatal("env variable NFS_BASE_PATH must be set so that this provisioner can identify itself")
        }

	nfsMode := "0700"
	nfsOwnerid := "65534"

	util.Infof("host=%s target=%s\n", nfsHost, nfsBasepath)

	mount, err := nfs.DialMount(nfsHost)
	if err != nil {
		glog.Fatal("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()

	auth := rpc.NewAuthUnix("hasselhoff", 0, 0)

	v, err := mount.Mount(nfsBasepath, auth.Auth())
	if err != nil {
		glog.Fatal("unable to mount volume: %v", err)
	}
        defer v.Close()

	return &nfsPathProvisioner{
		nfsHost:   nfsHost,
		nfsBasepath: nfsBasepath,
		nfsMode: nfsMode,
		nfsOwnerid: nfsOwnerid,
	}
}

var _ controller.Provisioner = &nfsPathProvisioner{}

// Provision creates a storage asset and returns a PV object representing it.
func (p *nfsPathProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
//	path := path.Join(p.pvDir, options.PVName)

//	if err := os.MkdirAll(path, 0777); err != nil {
//		return nil, err
//	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
//			Name: options.PVName,
			Name: "test",
			Annotations: map[string]string{
				"nfshost": p.nfsHost,
				"nfsbasepath": p.nfsBasepath,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server: p.nfsHost,
					Path: p.nfsBasepath,
				},
			},
		},
	}

	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *nfsPathProvisioner) Delete(volume *v1.PersistentVolume) error {
//	ann, ok := volume.Annotations["nfsPathProvisionerIdentity"]
//	if !ok {
//		return errors.New("identity annotation not found on PV")
//	}
//	if ann != p.identity {
//		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
//	}
//
//	path := path.Join(p.pvDir, volume.Name)
//	if err := os.RemoveAll(path); err != nil {
//		return err
//	}
//
	return nil
}

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")
	flag.Set("log_dir", "/dev/null")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	nfsPathProvisioner := NewNfsPathProvisioner()

	// Start the provision controller which will dynamically provision hostPath
	// PVs
	pc := controller.NewProvisionController(clientset, provisionerName, nfsPathProvisioner, serverVersion.GitVersion)
	pc.Run(wait.NeverStop)
}
