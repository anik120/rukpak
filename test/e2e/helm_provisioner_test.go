package e2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
	helm "github.com/operator-framework/rukpak/internal/provisioner/helm/types"
)

var _ = Describe("helm provisioner bundledeployment", func() {
	When("a BundleDeployment targets a valid Bundle", func() {
		var (
			bd  *rukpakv1alpha1.BundleDeployment
			ctx context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()

			bd = &rukpakv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ahoy-",
				},
				Spec: rukpakv1alpha1.BundleDeploymentSpec{
					ProvisionerClassName: helm.ProvisionerID,
					Template: &rukpakv1alpha1.BundleTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": "ahoy",
							},
						},
						Spec: rukpakv1alpha1.BundleSpec{
							ProvisionerClassName: helm.ProvisionerID,
							Source: rukpakv1alpha1.BundleSource{
								Type: rukpakv1alpha1.SourceTypeHTTP,
								HTTP: &rukpakv1alpha1.HTTPSource{
									URL: "https://github.com/helm/examples/releases/download/hello-world-0.1.0/hello-world-0.1.0.tgz",
								},
							},
						},
					},
				},
			}
			err := c.Create(ctx, bd)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("deleting the testing resources")
			Expect(c.Delete(ctx, bd)).To(BeNil())
		})

		It("should rollout the bundle contents successfully", func() {
			By("eventually writing a successful installation state back to the bundledeployment status")
			Eventually(func() (*metav1.Condition, error) {
				if err := c.Get(ctx, client.ObjectKeyFromObject(bd), bd); err != nil {
					return nil, err
				}
				if bd.Status.ActiveBundle == "" {
					return nil, fmt.Errorf("waiting for bundle name to be populated")
				}
				return meta.FindStatusCondition(bd.Status.Conditions, rukpakv1alpha1.TypeInstalled), nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *metav1.Condition) string { return c.Type }, Equal(rukpakv1alpha1.TypeInstalled)),
				WithTransform(func(c *metav1.Condition) metav1.ConditionStatus { return c.Status }, Equal(metav1.ConditionTrue)),
				WithTransform(func(c *metav1.Condition) string { return c.Reason }, Equal(rukpakv1alpha1.ReasonInstallationSucceeded)),
				WithTransform(func(c *metav1.Condition) string { return c.Message }, ContainSubstring("instantiated bundle")),
			))

			By("eventually install helm chart successfully")
			deployment := &appsv1.Deployment{}

			Eventually(func() (*appsv1.DeploymentCondition, error) {
				if err := c.Get(ctx, types.NamespacedName{Name: bd.GetName() + "-hello-world", Namespace: defaultSystemNamespace}, deployment); err != nil {
					return nil, err
				}
				for _, c := range deployment.Status.Conditions {
					if c.Type == appsv1.DeploymentAvailable {
						return &c, nil
					}
				}
				return nil, nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *appsv1.DeploymentCondition) appsv1.DeploymentConditionType { return c.Type }, Equal(appsv1.DeploymentAvailable)),
				WithTransform(func(c *appsv1.DeploymentCondition) corev1.ConditionStatus { return c.Status }, Equal(corev1.ConditionTrue)),
				WithTransform(func(c *appsv1.DeploymentCondition) string { return c.Reason }, Equal("MinimumReplicasAvailable")),
				WithTransform(func(c *appsv1.DeploymentCondition) string { return c.Message }, ContainSubstring("Deployment has minimum availability.")),
			))

			By("eventually recreate deleted resource in the helm chart")
			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: defaultSystemNamespace,
					Name:      bd.GetName() + "-hello-world",
				},
			}

			By("deleting resource in the helm chart")
			Expect(c.Delete(ctx, deployment)).To(BeNil())

			By("eventually recreate deleted resource in the helm chart")
			Eventually(func() (*appsv1.DeploymentCondition, error) {
				if err := c.Get(ctx, types.NamespacedName{Name: bd.GetName() + "-hello-world", Namespace: defaultSystemNamespace}, deployment); err != nil {
					return nil, err
				}
				for _, c := range deployment.Status.Conditions {
					if c.Type == appsv1.DeploymentAvailable {
						return &c, nil
					}
				}
				return nil, nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *appsv1.DeploymentCondition) appsv1.DeploymentConditionType { return c.Type }, Equal(appsv1.DeploymentAvailable)),
				WithTransform(func(c *appsv1.DeploymentCondition) corev1.ConditionStatus { return c.Status }, Equal(corev1.ConditionTrue)),
				WithTransform(func(c *appsv1.DeploymentCondition) string { return c.Reason }, Equal("MinimumReplicasAvailable")),
				WithTransform(func(c *appsv1.DeploymentCondition) string { return c.Message }, ContainSubstring("Deployment has minimum availability.")),
			))
		})
	})
	When("a BundleDeployment targets a Bundle with an invalid url", func() {
		var (
			bd  *rukpakv1alpha1.BundleDeployment
			ctx context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()

			bd = &rukpakv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ahoy-",
				},
				Spec: rukpakv1alpha1.BundleDeploymentSpec{
					ProvisionerClassName: helm.ProvisionerID,
					Template: &rukpakv1alpha1.BundleTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": "ahoy",
							},
						},
						Spec: rukpakv1alpha1.BundleSpec{
							ProvisionerClassName: helm.ProvisionerID,
							Source: rukpakv1alpha1.BundleSource{
								Type: rukpakv1alpha1.SourceTypeHTTP,
								HTTP: &rukpakv1alpha1.HTTPSource{
									URL: "https://github.com/helm/examples/releases/download/hello-world-0.1.0/xxx",
								},
							},
						},
					},
				},
			}
			err := c.Create(ctx, bd)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("deleting the testing resources")
			Expect(c.Delete(ctx, bd)).To(BeNil())
		})

		It("should fail rolling out the bundle contents", func() {
			By("eventually writing an installation state back to the bundledeployment status")
			Eventually(func() (*metav1.Condition, error) {
				if err := c.Get(ctx, client.ObjectKeyFromObject(bd), bd); err != nil {
					return nil, err
				}
				return meta.FindStatusCondition(bd.Status.Conditions, rukpakv1alpha1.TypeHasValidBundle), nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *metav1.Condition) string { return c.Type }, Equal(rukpakv1alpha1.TypeHasValidBundle)),
				WithTransform(func(c *metav1.Condition) metav1.ConditionStatus { return c.Status }, Equal(metav1.ConditionFalse)),
				WithTransform(func(c *metav1.Condition) string { return c.Reason }, Equal(rukpakv1alpha1.ReasonUnpackFailed)),
				WithTransform(func(c *metav1.Condition) string { return c.Message }, ContainSubstring(`unexpected status "404 Not Found"`)),
			))
		})
	})
	When("a BundleDeployment targets a Bundle with a none-tgz file url", func() {
		var (
			bd  *rukpakv1alpha1.BundleDeployment
			ctx context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()

			bd = &rukpakv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ahoy-",
				},
				Spec: rukpakv1alpha1.BundleDeploymentSpec{
					ProvisionerClassName: helm.ProvisionerID,
					Template: &rukpakv1alpha1.BundleTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": "ahoy",
							},
						},
						Spec: rukpakv1alpha1.BundleSpec{
							ProvisionerClassName: helm.ProvisionerID,
							Source: rukpakv1alpha1.BundleSource{
								Type: rukpakv1alpha1.SourceTypeHTTP,
								HTTP: &rukpakv1alpha1.HTTPSource{
									URL: "https://raw.githubusercontent.com/helm/examples/main/LICENSE",
								},
							},
						},
					},
				},
			}
			err := c.Create(ctx, bd)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("deleting the testing resources")
			Expect(c.Delete(ctx, bd)).To(BeNil())
		})

		It("should fail rolling out the bundle contents", func() {
			By("eventually writing an installation state back to the bundledeployment status")
			Eventually(func() (*metav1.Condition, error) {
				if err := c.Get(ctx, client.ObjectKeyFromObject(bd), bd); err != nil {
					return nil, err
				}
				return meta.FindStatusCondition(bd.Status.Conditions, rukpakv1alpha1.TypeHasValidBundle), nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *metav1.Condition) string { return c.Type }, Equal(rukpakv1alpha1.TypeHasValidBundle)),
				WithTransform(func(c *metav1.Condition) metav1.ConditionStatus { return c.Status }, Equal(metav1.ConditionFalse)),
				WithTransform(func(c *metav1.Condition) string { return c.Reason }, Equal(rukpakv1alpha1.ReasonUnpackFailed)),
				WithTransform(func(c *metav1.Condition) string { return c.Message }, ContainSubstring("gzip: invalid header")),
			))
		})
	})
	When("a BundleDeployment targets a Bundle with a none chart tgz url", func() {
		var (
			bd  *rukpakv1alpha1.BundleDeployment
			ctx context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()

			bd = &rukpakv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ahoy-",
				},
				Spec: rukpakv1alpha1.BundleDeploymentSpec{
					ProvisionerClassName: helm.ProvisionerID,
					Template: &rukpakv1alpha1.BundleTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": "ahoy",
							},
						},
						Spec: rukpakv1alpha1.BundleSpec{
							ProvisionerClassName: helm.ProvisionerID,
							Source: rukpakv1alpha1.BundleSource{
								Type: rukpakv1alpha1.SourceTypeHTTP,
								HTTP: &rukpakv1alpha1.HTTPSource{
									URL: "https://github.com/helm/examples/archive/refs/tags/hello-world-0.1.0.tar.gz",
								},
							},
						},
					},
				},
			}
			err := c.Create(ctx, bd)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			By("deleting the testing resources")
			Expect(c.Delete(ctx, bd)).To(BeNil())
		})

		It("should fail rolling out the bundle contents", func() {
			By("eventually writing an installation state back to the bundledeployment status")
			Eventually(func() (*metav1.Condition, error) {
				if err := c.Get(ctx, client.ObjectKeyFromObject(bd), bd); err != nil {
					return nil, err
				}
				return meta.FindStatusCondition(bd.Status.Conditions, rukpakv1alpha1.TypeHasValidBundle), nil
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(c *metav1.Condition) string { return c.Type }, Equal(rukpakv1alpha1.TypeHasValidBundle)),
				WithTransform(func(c *metav1.Condition) metav1.ConditionStatus { return c.Status }, Equal(metav1.ConditionFalse)),
				WithTransform(func(c *metav1.Condition) string { return c.Reason }, Equal(rukpakv1alpha1.ReasonUnpackFailed)),
				WithTransform(func(c *metav1.Condition) string { return c.Message }, ContainSubstring(" lint error: unable to check Chart.yaml file in chart")),
			))
		})
	})
})
