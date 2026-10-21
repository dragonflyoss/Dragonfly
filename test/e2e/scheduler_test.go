/*
 *     Copyright 2026 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint
	. "github.com/onsi/gomega"    //nolint

	"d7y.io/dragonfly/v2/test/e2e/util"
)

var _ = Describe("Schedulers Restarting", func() {
	Context("rollout restart", func() {
		var (
			testFile *util.File
			err      error
		)

		BeforeEach(func() {
			testFile, err = util.GetFileServer().GenerateFile(util.FileSize1MiB)
			Expect(err).NotTo(HaveOccurred())
			Expect(testFile).NotTo(BeNil())
		})

		AfterEach(func() {
			err = util.GetFileServer().DeleteFile(testFile.GetInfo())
			Expect(err).NotTo(HaveOccurred())
		})

		It("download should be ok after the schedulers restart", Label("scheduler", "restart"), func() {
			// Restart every scheduler, the statefulset recreates the pods with the
			// same names and new ips, so the clients must pick them up again.
			out, err := util.KubeCtlCommand("-n", util.DragonflyNamespace, "rollout", "restart", "statefulset", util.SchedulerStatefulSetName).CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			out, err = util.KubeCtlCommand("-n", util.DragonflyNamespace, "rollout", "status", "statefulset", util.SchedulerStatefulSetName, "--timeout=10m").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			clientPod, err := util.ClientExec()
			fmt.Println(err)
			Expect(err).NotTo(HaveOccurred())

			// Wait for the clients to refresh the schedulers and announce themselves
			// to the restarted schedulers.
			Eventually(func() error {
				out, err := clientPod.Command("sh", "-c", fmt.Sprintf("dfget %s --disable-back-to-source --output %s", testFile.GetDownloadURL(), testFile.GetOutputPath())).CombinedOutput()
				fmt.Println(string(out), err)
				return err
			}, 5*time.Minute, 15*time.Second).Should(Succeed())

			// The download above proves one scheduler tracks the seed peers again,
			// wait for the rest: the seed peers discover a restarted scheduler within
			// the dynconfig refresh interval and announce within the announce interval,
			// and the schedulers of the compatibility tests rebuild the hashring of
			// seed peers every 30 seconds after that.
			time.Sleep(90 * time.Second)

			sha256sum, err := util.CalculateSha256ByTaskID([]*util.PodExec{clientPod}, testFile.GetTaskID())
			Expect(err).NotTo(HaveOccurred())
			Expect(testFile.GetSha256()).To(Equal(sha256sum))

			sha256sum, err = util.CalculateSha256ByOutput([]*util.PodExec{clientPod}, testFile.GetOutputPath())
			Expect(err).NotTo(HaveOccurred())
			Expect(testFile.GetSha256()).To(Equal(sha256sum))
		})
	})
})
