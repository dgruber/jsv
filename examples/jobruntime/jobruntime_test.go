package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"os/exec"

	"github.com/dgruber/jsv/test/jsvserver"
)

var binaryName = "./jobruntime_test"

// Build the binary once for all test nodes.
var _ = BeforeSuite(func() {
	err := exec.Command("go", "build", "-o", binaryName, "jobruntime.go").Run()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Jobruntime", func() {
	var server *jsvserver.JSVTestServer

	BeforeEach(func() {
		var err error
		server, err = jsvserver.NewJSVTestServer(binaryName)
		Expect(err).ToNot(HaveOccurred())

		err = server.Start()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := server.Stop()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when job has requested the queue long.q", func() {

		It("should reject the job since no hard runtime limit is requested", func() {
			result, err := server.SendJob(&jsvserver.JobSpec{
				Group:   "test",
				CmdName: "myjob.sh",
				CmdArgs: 0,
				Params: map[string]string{
					"q_hard": "long.q",
				},
				Environment: map[string]string{
					"PATH": "/usr/bin:/bin",
					"USER": "testuser",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.State).To(Equal("REJECT"))
			Expect(result.Message).To(ContainSubstring("No hard runtime limit requested (h_rt)"))
		})

		It("should accept the job since a hard runtime limit is requested", func() {
			result, err := server.SendJob(&jsvserver.JobSpec{
				Group:   "test",
				CmdName: "myjob.sh",
				CmdArgs: 0,
				Params: map[string]string{
					"q_hard": "long.q",
					"l_hard": "h_rt=600",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.State).To(Equal("ACCEPT"))
		})

		It("should modify the runtime limit to 10 minutes", func() {
			result, err := server.SendJob(&jsvserver.JobSpec{
				Group:   "test",
				CmdName: "myjob.sh",
				CmdArgs: 0,
				Params: map[string]string{
					"q_hard": "long.q",
					"l_hard": "h_rt=500",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.State).To(Equal("CORRECT"))
			Expect(result.Message).To(ContainSubstring("Runtime limit was increased to 10 minutes"))
			Expect(result.ModifiedParams["l_hard"]).To(Equal("h_rt=600"))
		})

	})
})
