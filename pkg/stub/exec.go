package stub

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net"
	"os"
)

var (
	kubeClient      *kubernetes.Clientset
	inClusterConfig *rest.Config
)

func init() {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}

	var err error
	inClusterConfig, err = rest.InClusterConfig()

	if err != nil {
		panic(err)
	}

	kubeClient = kubernetes.NewForConfigOrDie(inClusterConfig)
}

// execCommand executes the given command inside the specified container remotely
func execCommand(namespace, podName string, stdinReader io.Reader, container *v1.Container, command ...string) (string, error) {

	execReq := kubeClient.CoreV1().RESTClient().Post()
	execReq = execReq.Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")

	execReq.VersionedParams(&v1.PodExecOptions{
		Container: container.Name,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
		Stdin:     stdinReader != nil,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(inClusterConfig, "POST", execReq.URL())

	if err != nil {
		logrus.Errorf("Creating remote command executor failed: %v", err)
		return "", err
	}
	//输出流缓冲区
	stdOut := bytes.Buffer{}
	//错误输出流缓冲区
	stdErr := bytes.Buffer{}

	logrus.Debugf("Executing command '%v' in namespace='%s', pod='%s', container='%s'", command, namespace, podName, container.Name)
	//执行指令 Stdout 是 POD 的标准输出流，Stdin 是 POD 的标准输入流，Stderr 是标准错误输出流
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: bufio.NewWriter(&stdOut),
		Stderr: bufio.NewWriter(&stdErr),
		//这里是是需要往 POD 中传入的内容，使用 POD POD 标准输入流进行传输，在 POD 内部处理接收标准输入流就可以获取需要传输的文件
		Stdin:  stdinReader,
		Tty:    false,
	})

	logrus.Debugf("Command stderr: %s", stdErr.String())
	logrus.Debugf("Command stdout: %s", stdOut.String())

	if err != nil {
		logrus.Infof("Executing command failed with: %v", err)

		return "", err
	}

	logrus.Debug("Command succeeded.")
	if stdErr.Len() > 0 {
		return "", fmt.Errorf("stderr: %v", stdErr.String())
	}

	return stdOut.String(), nil

}
