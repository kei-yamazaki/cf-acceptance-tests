package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = Describe("Crashing", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("a continuously crashing app", func() {
		BeforeEach(func() {
			if config.Backend != "diego" {
				Skip(`Skipping this test because config.Backend is not set to 'diego'
NOTE: Ensure your platform is running Diego before enabling this test`)
			}
		})

		It("emits crash events and reports as 'crashed' after enough crashes", func() {
			Expect(cf.Cf(
				"push",
				appName,
				"-c", "/bin/false",
				"--no-start",
				"-b", config.RubyBuildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", config.AppsDomain,
			).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
			}, DEFAULT_TIMEOUT).Should(MatchRegexp("[eE]xited"))

			Eventually(cf.Cf("app", appName), DEFAULT_TIMEOUT).Should(Say("crashed"))
		})
	})

	Context("the app crashes", func() {
		BeforeEach(func() {
			Expect(cf.Cf(
				"push",
				appName,
				"--no-start",
				"-b", config.RubyBuildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", config.AppsDomain,
			).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		It("shows crash events", func() {
			helpers.CurlApp(appName, "/sigterm/KILL")

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
			}, DEFAULT_TIMEOUT).Should(MatchRegexp("[eE]xited"))
		})

		It("recovers", func() {
			id := helpers.CurlApp(appName, "/id")
			helpers.CurlApp(appName, "/sigterm/KILL")

			Eventually(func() string {
				return helpers.CurlApp(appName, "/id")
			}, DEFAULT_TIMEOUT).Should(Not(Equal(id)))
		})
	})
})
