package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ncsibra/operator/api/v1alpha1"
)

var _ = Describe("Sensitive controller", func() {
	const (
		SensitiveResourceName      = "sensitive-test"
		SensitiveResourceNamespace = "default"
		SensitiveResourceKey       = "password"
		SensitiveResourceValue     = "test-password"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	BeforeEach(func() {
		err := k8sClient.Delete(ctx, &v1alpha1.Sensitive{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SensitiveResourceName,
				Namespace: SensitiveResourceNamespace,
			},
		})
		if err != nil && !errors.IsNotFound(err) {
			Ω(err).ShouldNot(HaveOccurred(), "unable to cleanup Sensitive resource")
		}

		err = k8sClient.Delete(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SensitiveResourceName,
				Namespace: SensitiveResourceNamespace,
			},
		})
		if err != nil && !errors.IsNotFound(err) {
			Ω(err).ShouldNot(HaveOccurred(), "unable to cleanup Secret resource")
		}
	})

	Context("When creating a Sensitive resource", func() {
		It("Should create a matching Secret resource", func() {
			By("Creating a new Sensitive resource", func() {
				ctx := context.Background()

				sensitive := &v1alpha1.Sensitive{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "test.origoss.com/v1alpha1",
						Kind:       "Sensitive",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      SensitiveResourceName,
						Namespace: SensitiveResourceNamespace,
					},
					Spec: v1alpha1.SensitiveSpec{
						Key:   SensitiveResourceKey,
						Value: SensitiveResourceValue,
					},
				}

				Expect(k8sClient.Create(ctx, sensitive)).Should(Succeed())
			})

			By("Checking the Secret created", func() {
				var secret v1.Secret

				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: SensitiveResourceName, Namespace: SensitiveResourceNamespace}, &secret)
					if err != nil {
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())

				secretValue, ok := secret.Data[SensitiveResourceKey]

				Expect(ok).Should(BeTrue(), "Secret does not contains the key from Sensitive resource, key: "+SensitiveResourceKey)
				Expect(string(secretValue)).Should(Equal(SensitiveResourceValue))
				Expect(secret.GetOwnerReferences()).Should(HaveLen(1))

				ownerReference := secret.GetOwnerReferences()[0]

				Expect(ownerReference.Kind).Should(Equal("Sensitive"))
				Expect(ownerReference.Name).Should(Equal(SensitiveResourceName))
			})
		})
	})

	Context("When deleting a Sensitive resource", func() {
		It("Should delete the matching Secret resource", func() {

			var sensitive v1alpha1.Sensitive

			By("Creating a new Sensitive resource", func() {
				ctx := context.Background()

				sensitive = v1alpha1.Sensitive{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "test.origoss.com/v1alpha1",
						Kind:       "Sensitive",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      SensitiveResourceName,
						Namespace: SensitiveResourceNamespace,
					},
					Spec: v1alpha1.SensitiveSpec{
						Key:   SensitiveResourceKey,
						Value: SensitiveResourceValue,
					},
				}

				Expect(k8sClient.Create(ctx, &sensitive)).Should(Succeed())
			})

			var secret v1.Secret

			By("Checking the Secret created", func() {
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: SensitiveResourceName, Namespace: SensitiveResourceNamespace}, &secret)
					if err != nil {
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())
			})

			By("Deleting the Sensitive resource", func() {
				Expect(k8sClient.Delete(ctx, &sensitive)).Should(Succeed())

				var deletedSensitive v1alpha1.Sensitive

				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: SensitiveResourceName, Namespace: SensitiveResourceNamespace}, &deletedSensitive)
					if errors.IsNotFound(err) {
						return true
					}

					return false
				}, timeout, interval).Should(BeTrue())
			})

			By("Checking the Secret deleted too", func() {
				// envtest won't delete the secret by owner reference
				// https://book.kubebuilder.io/reference/envtest.html#testing-considerations

				trueVal := true

				expectedOwnerReference := metav1.OwnerReference{
					Kind:               "Sensitive",
					APIVersion:         "test.origoss.com/v1alpha1",
					UID:                sensitive.UID,
					Name:               sensitive.Name,
					Controller:         &trueVal,
					BlockOwnerDeletion: &trueVal,
				}

				Expect(secret.OwnerReferences).Should(ContainElement(expectedOwnerReference))
			})
		})
	})
})
