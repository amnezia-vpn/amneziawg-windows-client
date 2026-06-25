/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2022 WireGuard LLC. All Rights Reserved.
 */

package updater

const (
	releasePublicKeyBase64 = "RWTIu7lL938k14FQAOYx3R4MaIKLv6CmaxBI3G8zV7tbjpmqFY38l2r3"
	updateServerHost       = "updates.amnezia.org"
	updateServerPort       = 443
	updateServerUseHttps   = true
	latestVersionPath      = "/amneziawg/windows-client/latest.sig"
	msiPath                = "/amneziawg/windows-client/%s"
	msiArchPrefix          = "amneziawg-%s-"
	msiSuffix              = ".msi"
)
