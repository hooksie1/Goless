package controllers

import (
	"context"
	"time"

	functionv1beta1 "github.com/hooksie1/goless/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Function controller", func() {
	const (
		FunctionName      = "test-function"
		FunctionNamespace = "default"

		timeout      = time.Second * 10
		duration     = time.Second * 10
		interval     = time.Millisecond * 250
		serverPort   = 9000
		service      = "test"
		functionData = `
		package handlers
		import (
			"net/http"
		)

		func Handler(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test succussful!"))
		}
		`
	)

	Context("Function", func() {
		It("Should create Successfully", func() {
			By("By creating a new function")
			ctx := context.Background()
			function := functionv1beta1.Function{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "goless.io/v1beta1",
					Kind:       "Function",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      FunctionName,
					Namespace: FunctionNamespace,
				},
				Spec: functionv1beta1.FunctionSpec{
					Service:    service,
					ServerPort: serverPort,
					Function:   functionData,
				},
			}
			Expect(k8sClient.Create(ctx, &function)).Should(Succeed())
			functionLookupKey := types.NamespacedName{Name: FunctionName, Namespace: FunctionNamespace}
			createdFunction := &functionv1beta1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, functionLookupKey, createdFunction)
				if err != nil {
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			updated := &functionv1beta1.Function{}
			Expect(k8sClient.Get(ctx, functionLookupKey, updated)).Should(Succeed())
			function.Spec.ServerPort = 8000
			Expect(k8sClient.Update(context.Background(), updated)).Should(Succeed())

			By("Expecting to delete function")
			Eventually(func() error {
				f := &functionv1beta1.Function{}
				k8sClient.Get(context.Background(), functionLookupKey, f)
				return k8sClient.Delete(context.Background(), f)
			}, timeout, interval).Should(Succeed())
		})
	})
})
