package main

import (
	"context"
	"runtime"

	stub "github.com/banzaicloud/prometheus-jmx-exporter-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
	"os"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()
	namespace := os.Getenv("OPERATOR_NAMESPACE")
        //监控 crd 自定义资源变化
	sdk.Watch("banzaicloud.com/v1alpha1", "PrometheusJmxExporter", namespace, 0) 
	//监控 pod 资源变化   自定义资源变化主要是监控选择lable变化，pod 主要是如果有新的pod创建，应该检查是否嵌入agent
	sdk.Watch("v1", "Pod", namespace, 0)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
