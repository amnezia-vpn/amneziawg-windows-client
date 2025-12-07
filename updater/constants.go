/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2022 WireGuard LLC. All Rights Reserved.
 */

package updater

const (
	releasePublicKeyBase64 = "RWTWrwVyWyYJzah2mvcm/mk3RGR7xHaAIznKg2CwB+geUS81MQSoT9UO"
	updateServerHost       = "romikb.ru"
	updateServerPort       = 443
	updateServerUseHttps   = true
	latestVersionPath      = "/windows-client/latest.sig"
	msiPath                = "/windows-client/%s"
	msiArchPrefix          = "amneziawg-%s-"
	msiSuffix              = ".msi"
)
