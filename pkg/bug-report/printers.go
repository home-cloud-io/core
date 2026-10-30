package bugreport

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/home-cloud-io/core/pkg/kube"
)

type Printer struct {
	Root         string
	SkippedTypes []string
	Client  kube.Client
	Printer printers.ResourcePrinter
}

func (p Printer) Print(ctx context.Context) error {

	err := os.RemoveAll(p.Root)
	if err != nil {
		return err
	}

	err = os.MkdirAll(p.Root, os.ModePerm)
	if err != nil {
		return err
	}

	err = p.printNodes(ctx)
	if err != nil {
		return err
	}

	err = p.printNamespaced(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p Printer) Archive() error {
	mw, err := os.Create("bug-report.tar.gz")
	if err != nil {
		return err
	}
	defer mw.Close()
	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = filepath.Walk(p.Root, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(file, p.Root, "", -1), string(filepath.Separator))
		header.Size = fi.Size()
		header.Mode = int64(fi.Mode())
		header.ModTime = fi.ModTime()
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (p Printer) printNodes(ctx context.Context) error {
	list, err := p.Client.Clientset.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(p.Root, "nodes.yaml"))
	if err != nil {
		return err
	}
	for _, node := range list.Items {
		node.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Node",
		})
		err = p.Printer.PrintObj(node.DeepCopy(), f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Printer) printNamespaced(ctx context.Context) error {
	list, err := p.Client.Clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range list.Items {
		err = os.MkdirAll(filepath.Join(p.Root, ns.Name), os.ModePerm)
		if err != nil {
			return err
		}

		f, err := os.Create(filepath.Join(p.Root, ns.Name, fmt.Sprintf("namespace.yaml")))
		if err != nil {
			return err
		}
		ns.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Namespace",
		})
		err = p.Printer.PrintObj(ns.DeepCopy(), f)
		if err != nil {
			return err
		}

		if !slices.Contains(p.SkippedTypes, "pods") {
			err := p.printPods(ctx, ns.Name)
			if err != nil {
				return err
			}
		}

		// print configured namespaced types
		for kind, t := range typesMap {
			if slices.Contains(p.SkippedTypes, kind) {
				continue
			}
			err := p.printUnstructuredList(ctx, kind, ns.Name, t)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p Printer) printPods(ctx context.Context, namespace string) error {
	list := &corev1.PodList{}
	err := p.Client.Client.List(ctx, list, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	}

	f, err := os.Create(filepath.Join(p.Root, namespace, "pods.yaml"))
	if err != nil {
		return err
	}
	for _, pod := range list.Items {
		pod.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Pod",
		})
		err = p.Printer.PrintObj(pod.DeepCopy(), f)
		if err != nil {
			return err
		}
	}

	err = p.printPodLogs(ctx, namespace, list.Items)
	if err != nil {
		return err
	}

	return nil
}

func (p Printer) printPodLogs(ctx context.Context, namespace string, pods []corev1.Pod) error {
	err := os.MkdirAll(filepath.Join(p.Root, namespace, "logs"), os.ModePerm)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		err := os.MkdirAll(filepath.Join(p.Root, pod.Namespace, "logs", pod.Name), os.ModePerm)
		if err != nil {
			return err
		}

		for _, container := range pod.Spec.Containers {
			f, err := os.Create(filepath.Join(p.Root, pod.Namespace, "logs", pod.Name, fmt.Sprintf("%s.log", container.Name)))
			if err != nil {
				return err
			}

			req := p.Client.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Timestamps: true,
				Container:  container.Name,
			})
			logs, err := req.Stream(ctx)
			if err != nil {
				return err
			}
			defer logs.Close()

			_, err = io.Copy(f, logs)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (p Printer) printUnstructuredList(ctx context.Context, kind string, namespace string, t typeConfig) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(t.gvk)
	err := p.Client.Client.List(ctx, list, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	// ignore empty lists
	if len(list.Items) == 0 {
		return nil
	}

	f, err := os.Create(filepath.Join(p.Root, namespace, fmt.Sprintf("%s.yaml", kind)))
	if err != nil {
		return err
	}

	for _, o := range list.Items {
		// if set, filter before printing
		if t.filter != nil {
			t.filter(&o)
		}

		err = p.Printer.PrintObj(o.DeepCopyObject(), f)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	return nil

}
