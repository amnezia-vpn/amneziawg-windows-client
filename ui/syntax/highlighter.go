/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2022 WireGuard LLC. All Rights Reserved.
 *
 * This is a direct translation of the original C, and for that reason, it's pretty unusual Go code:
 * https://git.zx2c4.com/wireguard-tools/tree/contrib/highlighter/highlighter.c
 */

package syntax

import (
	"strconv"
	"unsafe"

	"github.com/amnezia-vpn/amneziawg-go/device"
)

type highlight int

const (
	highlightSection highlight = iota
	highlightField
	highlightPrivateKey
	highlightPublicKey
	highlightPresharedKey
	highlightIP
	highlightCidr
	highlightHost
	highlightPort
	highlightMTU
	highlightKeepalive
	highlightComment
	highlightDelimiter
	highlightTable
	highlightCmd
	highlightJc
	highlightJmin
	highlightJmax
	highlightS1
	highlightS2
	highlightH1
	highlightH2
	highlightH3
	highlightH4
	highlightWarning
	highlightError
)

func validateHighlight(isValid bool, t highlight) highlight {
	if isValid {
		return t
	}
	return highlightError
}

type highlightSpan struct {
	t   highlight
	s   int
	len int
}

func isDecimal(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexadecimal(c byte) bool {
	return isDecimal(c) || (c|32) >= 'a' && (c|32) <= 'f'
}

func isAlphabet(c byte) bool {
	return (c|32) >= 'a' && (c|32) <= 'z'
}

type stringSpan struct {
	s   *byte
	len int
}

func (s stringSpan) at(i int) *byte {
	return (*byte)(unsafe.Add(unsafe.Pointer(s.s), uintptr(i)))
}

func (s stringSpan) isSame(c string) bool {
	if s.len != len(c) {
		return false
	}
	cb := ([]byte)(c)
	for i := 0; i < s.len; i++ {
		if *s.at(i) != cb[i] {
			return false
		}
	}
	return true
}

func (s stringSpan) isCaselessSame(c string) bool {
	if s.len != len(c) {
		return false
	}
	cb := ([]byte)(c)
	for i := 0; i < s.len; i++ {
		a := *s.at(i)
		b := cb[i]
		if a-'a' < 26 {
			a &= 95
		}
		if b-'a' < 26 {
			b &= 95
		}
		if a != b {
			return false
		}
	}
	return true
}

func (s stringSpan) isValidKey() bool {
	if s.len != 44 || *s.at(43) != '=' {
		return false
	}
	for i := 0; i < 42; i++ {
		if !isDecimal(*s.at(i)) && !isAlphabet(*s.at(i)) && *s.at(i) != '/' && *s.at(i) != '+' {
			return false
		}
	}
	switch *s.at(42) {
	case 'A', 'E', 'I', 'M', 'Q', 'U', 'Y', 'c', 'g', 'k', 'o', 's', 'w', '4', '8', '0':
		return true
	}
	return false
}

func (s stringSpan) isValidHostname() bool {
	numDigit := 0
	numEntity := s.len
	if s.len > 63 || s.len == 0 {
		return false
	}
	if *s.s == '-' || *s.at(s.len - 1) == '-' {
		return false
	}
	if *s.s == '.' || *s.at(s.len - 1) == '.' {
		return false
	}
	for i := 0; i < s.len; i++ {
		if isDecimal(*s.at(i)) {
			numDigit++
			continue
		}
		if *s.at(i) == '.' {
			numEntity--
			continue
		}
		if !isAlphabet(*s.at(i)) && *s.at(i) != '-' {
			return false
		}
		if i != 0 && *s.at(i) == '.' && *s.at(i - 1) == '.' {
			return false
		}
	}
	return numDigit != numEntity
}

func (s stringSpan) isValidIPv4() bool {
	pos := 0
	for i := 0; i < 4 && pos < s.len; i++ {
		val := 0
		j := 0
		for ; j < 3 && pos+j < s.len && isDecimal(*s.at(pos + j)); j++ {
			val = 10*val + int(*s.at(pos + j)-'0')
		}
		if j == 0 || j > 1 && *s.at(pos) == '0' || val > 255 {
			return false
		}
		if pos+j == s.len && i == 3 {
			return true
		}
		if *s.at(pos + j) != '.' {
			return false
		}
		pos += j + 1
	}
	return false
}

func (s stringSpan) isValidIPv6() bool {
	if s.len < 2 {
		return false
	}
	pos := 0
	if *s.at(0) == ':' {
		if *s.at(1) != ':' {
			return false
		}
		pos = 1
	}
	if *s.at(s.len - 1) == ':' && *s.at(s.len - 2) != ':' {
		return false
	}
	seenColon := false
	for i := 0; pos < s.len; i++ {
		if *s.at(pos) == ':' && !seenColon {
			seenColon = true
			pos++
			if pos == s.len {
				break
			}
			if i == 7 {
				return false
			}
			continue
		}
		j := 0
		for ; ; j++ {
			if j < 4 && pos+j < s.len && isHexadecimal(*s.at(pos + j)) {
				continue
			}
			break
		}
		if j == 0 {
			return false
		}
		if pos+j == s.len && (seenColon || i == 7) {
			break
		}
		if i == 7 {
			return false
		}
		if *s.at(pos + j) != ':' {
			if *s.at(pos + j) != '.' || i < 6 && !seenColon {
				return false
			}
			return stringSpan{s.at(pos), s.len - pos}.isValidIPv4()
		}
		pos += j + 1
	}
	return true
}

func (s stringSpan) isValidUint(supportHex bool, min, max uint64) bool {
	// Bound this around 32 bits, so that we don't have to write overflow logic.
	if s.len > 10 || s.len == 0 {
		return false
	}
	val := uint64(0)
	if supportHex && s.len > 2 && *s.s == '0' && *s.at(1) == 'x' {
		for i := 2; i < s.len; i++ {
			if *s.at(i)-'0' < 10 {
				val = 16*val + uint64(*s.at(i)-'0')
			} else if (*s.at(i))|32-'a' < 6 {
				val = 16*val + uint64((*s.at(i)|32)-'a'+10)
			} else {
				return false
			}
		}
	} else {
		for i := 0; i < s.len; i++ {
			if !isDecimal(*s.at(i)) {
				return false
			}
			val = 10*val + uint64(*s.at(i)-'0')
		}
	}
	return val <= max && val >= min
}

func (s stringSpan) isValidPort() bool {
	return s.isValidUint(false, 0, 65535)
}

func (s stringSpan) isValidMTU() bool {
	return s.isValidUint(false, 576, 65535)
}

func (s stringSpan) isValidTable() bool {
	return s.isSame("off") || s.isSame("auto") || s.isSame("main") || s.isValidUint(false, 0, (1<<32)-1)
}

func (s stringSpan) isValidPersistentKeepAlive() bool {
	if s.isSame("off") {
		return true
	}
	return s.isValidUint(false, 0, 65535)
}

// It's probably not worthwhile to try to validate a bash expression. So instead we just demand non-zero length.
func (s stringSpan) isValidPrePostUpDown() bool {
	return s.len != 0
}

func (s stringSpan) isValidScope() bool {
	if s.len > 64 || s.len == 0 {
		return false
	}
	for i := 0; i < s.len; i++ {
		if isAlphabet(*s.at(i)) && !isDecimal(*s.at(i)) && *s.at(i) != '_' && *s.at(i) != '=' && *s.at(i) != '+' && *s.at(i) != '.' && *s.at(i) != '-' {
			return false
		}
	}
	return true
}

func (s stringSpan) isValidEndpoint() bool {
	if s.len == 0 {
		return false
	}
	if *s.s == '[' {
		seenScope := false
		hostspan := stringSpan{s.at(1), 0}
		for i := 1; i < s.len; i++ {
			if *s.at(i) == '%' {
				if seenScope {
					return false
				}
				seenScope = true
				if !hostspan.isValidIPv6() {
					return false
				}
				hostspan = stringSpan{s.at(i + 1), 0}
			} else if *s.at(i) == ']' {
				if seenScope {
					if !hostspan.isValidScope() {
						return false
					}
				} else if !hostspan.isValidIPv6() {
					return false
				}
				if i == s.len-1 || *s.at((i + 1)) != ':' {
					return false
				}
				return stringSpan{s.at(i + 2), s.len - i - 2}.isValidPort()
			} else {
				hostspan.len++
			}
		}
		return false
	}
	for i := 0; i < s.len; i++ {
		if *s.at(i) == ':' {
			host := stringSpan{s.s, i}
			port := stringSpan{s.at(i + 1), s.len - i - 1}
			return port.isValidPort() && (host.isValidIPv4() || host.isValidHostname())
		}
	}
	return false
}

func (s stringSpan) isValidNetwork() bool {
	for i := 0; i < s.len; i++ {
		if *s.at(i) == '/' {
			ip := stringSpan{s.s, i}
			cidr := stringSpan{s.at(i + 1), s.len - i - 1}
			cidrval := uint16(0)
			if cidr.len > 3 || cidr.len == 0 {
				return false
			}
			for j := 0; j < cidr.len; j++ {
				if !isDecimal(*cidr.at(j)) {
					return false
				}
				cidrval = 10*cidrval + uint16(*cidr.at(j)-'0')
			}
			if ip.isValidIPv4() {
				return cidrval <= 32
			} else if ip.isValidIPv6() {
				return cidrval <= 128
			}
			return false
		}
	}
	return s.isValidIPv4() || s.isValidIPv6()
}

type field int32

const (
	fieldInterfaceSection field = iota
	fieldPrivateKey
	fieldListenPort
	fieldAddress
	fieldDNS
	fieldMTU
	fieldTable
	fieldPreUp
	fieldPostUp
	fieldPreDown
	fieldPostDown
	fieldJc
	fieldJmin
	fieldJmax
	fieldS1
	fieldS2
	fieldH1
	fieldH2
	fieldH3
	fieldH4
	fieldPeerSection
	fieldPublicKey
	fieldPresharedKey
	fieldAllowedIPs
	fieldEndpoint
	fieldPersistentKeepalive
	fieldInvalid
)

func sectionForField(t field) field {
	if t > fieldInterfaceSection && t < fieldPeerSection {
		return fieldInterfaceSection
	}
	if t > fieldPeerSection && t < fieldInvalid {
		return fieldPeerSection
	}
	return fieldInvalid
}

func (s stringSpan) field() field {
	switch {
	case s.isCaselessSame("PrivateKey"):
		return fieldPrivateKey
	case s.isCaselessSame("ListenPort"):
		return fieldListenPort
	case s.isCaselessSame("Address"):
		return fieldAddress
	case s.isCaselessSame("DNS"):
		return fieldDNS
	case s.isCaselessSame("MTU"):
		return fieldMTU
	case s.isCaselessSame("Table"):
		return fieldTable
	case s.isCaselessSame("PublicKey"):
		return fieldPublicKey
	case s.isCaselessSame("PresharedKey"):
		return fieldPresharedKey
	case s.isCaselessSame("AllowedIPs"):
		return fieldAllowedIPs
	case s.isCaselessSame("Endpoint"):
		return fieldEndpoint
	case s.isCaselessSame("PersistentKeepalive"):
		return fieldPersistentKeepalive
	case s.isCaselessSame("PreUp"):
		return fieldPreUp
	case s.isCaselessSame("PostUp"):
		return fieldPostUp
	case s.isCaselessSame("PreDown"):
		return fieldPreDown
	case s.isCaselessSame("PostDown"):
		return fieldPostDown
	case s.isCaselessSame("Jc"):
		return fieldJc
	case s.isCaselessSame("Jmin"):
		return fieldJmin
	case s.isCaselessSame("Jmax"):
		return fieldJmax
	case s.isCaselessSame("S1"):
		return fieldS1
	case s.isCaselessSame("S2"):
		return fieldS2
	case s.isCaselessSame("H1"):
		return fieldH1
	case s.isCaselessSame("H2"):
		return fieldH2
	case s.isCaselessSame("H3"):
		return fieldH3
	case s.isCaselessSame("H4"):
		return fieldH4
	}
	return fieldInvalid
}

func (s stringSpan) sectionType() field {
	switch {
	case s.isCaselessSame("[Peer]"):
		return fieldPeerSection
	case s.isCaselessSame("[Interface]"):
		return fieldInterfaceSection
	}
	return fieldInvalid
}

type highlightSpanArray []highlightSpan

func (hsa *highlightSpanArray) append(o *byte, s stringSpan, t highlight) {
	if s.len == 0 {
		return
	}
	*hsa = append(*hsa, highlightSpan{t, int((uintptr(unsafe.Pointer(s.s))) - (uintptr(unsafe.Pointer(o)))), s.len})
}

func (hsa *highlightSpanArray) highlightMultivalueValue(parent, s stringSpan, section field) {
	switch section {
	case fieldDNS:
		if s.isValidIPv4() || s.isValidIPv6() {
			hsa.append(parent.s, s, highlightIP)
		} else if s.isValidHostname() {
			hsa.append(parent.s, s, highlightHost)
		} else {
			hsa.append(parent.s, s, highlightError)
		}
	case fieldAddress, fieldAllowedIPs:
		if !s.isValidNetwork() {
			hsa.append(parent.s, s, highlightError)
			break
		}
		slash := 0
		for ; slash < s.len; slash++ {
			if *s.at(slash) == '/' {
				break
			}
		}
		if slash == s.len {
			hsa.append(parent.s, s, highlightIP)
		} else {
			hsa.append(parent.s, stringSpan{s.s, slash}, highlightIP)
			hsa.append(parent.s, stringSpan{s.at(slash), 1}, highlightDelimiter)
			hsa.append(parent.s, stringSpan{s.at(slash + 1), s.len - slash - 1}, highlightCidr)
		}
	default:
		hsa.append(parent.s, s, highlightError)
	}
}

func (hsa *highlightSpanArray) highlightMultivalue(parent, s stringSpan, section field) {
	currentSpan := stringSpan{s.s, 0}
	lenAtLastSpace := 0
	for i := 0; i < s.len; i++ {
		if *s.at(i) == ',' {
			currentSpan.len = lenAtLastSpace
			hsa.highlightMultivalueValue(parent, currentSpan, section)
			hsa.append(parent.s, stringSpan{s.at(i), 1}, highlightDelimiter)
			lenAtLastSpace = 0
			currentSpan = stringSpan{s.at(i + 1), 0}
		} else if *s.at(i) == ' ' || *s.at(i) == '\t' {
			if s.at(i) == currentSpan.s && currentSpan.len == 0 {
				currentSpan.s = currentSpan.at(1)
			} else {
				currentSpan.len++
			}
		} else {
			currentSpan.len++
			lenAtLastSpace = currentSpan.len
		}
	}
	currentSpan.len = lenAtLastSpace
	if currentSpan.len != 0 {
		hsa.highlightMultivalueValue(parent, currentSpan, section)
	} else if (*hsa)[len(*hsa)-1].t == highlightDelimiter {
		(*hsa)[len(*hsa)-1].t = highlightError
	}
}

func (hsa *highlightSpanArray) highlightValue(parent, s stringSpan, section field) {
	switch section {
	case fieldPrivateKey:
		hsa.append(parent.s, s, validateHighlight(s.isValidKey(), highlightPrivateKey))
	case fieldPublicKey:
		hsa.append(parent.s, s, validateHighlight(s.isValidKey(), highlightPublicKey))
	case fieldPresharedKey:
		hsa.append(parent.s, s, validateHighlight(s.isValidKey(), highlightPresharedKey))
	case fieldMTU:
		hsa.append(parent.s, s, validateHighlight(s.isValidMTU(), highlightMTU))
	case fieldTable:
		hsa.append(parent.s, s, validateHighlight(s.isValidTable(), highlightTable))
	case fieldPreUp, fieldPostUp, fieldPreDown, fieldPostDown:
		hsa.append(parent.s, s, validateHighlight(s.isValidPrePostUpDown(), highlightCmd))
	case fieldListenPort:
		hsa.append(parent.s, s, validateHighlight(s.isValidPort(), highlightPort))
	case fieldPersistentKeepalive:
		hsa.append(parent.s, s, validateHighlight(s.isValidPersistentKeepAlive(), highlightKeepalive))
	case fieldEndpoint:
		if !s.isValidEndpoint() {
			hsa.append(parent.s, s, highlightError)
			break
		}
		colon := s.len
		for colon > 0 {
			colon--
			if *s.at(colon) == ':' {
				break
			}
		}
		hsa.append(parent.s, stringSpan{s.s, colon}, highlightHost)
		hsa.append(parent.s, stringSpan{s.at(colon), 1}, highlightDelimiter)
		hsa.append(parent.s, stringSpan{s.at(colon + 1), s.len - colon - 1}, highlightPort)
	case fieldAddress, fieldDNS, fieldAllowedIPs:
		hsa.highlightMultivalue(parent, s, section)
	case fieldJc:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 65_535), highlightJc))
	case fieldJmin:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 65_535), highlightJmin))
	case fieldJmax:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 65_535), highlightJmax))
	case fieldS1:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 65_535), highlightS1))
	case fieldS2:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 65_535), highlightS2))
	case fieldH1:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 2_147_483_647), highlightH1))
	case fieldH2:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 2_147_483_647), highlightH2))
	case fieldH3:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 2_147_483_647), highlightH3))
	case fieldH4:
		hsa.append(parent.s, s, validateHighlight(s.isValidUint(false, 0, 2_147_483_647), highlightH4))
	default:
		hsa.append(parent.s, s, highlightError)
	}
}

func highlightConfig(config string) []highlightSpan {
	var ret highlightSpanArray
	b := append([]byte(config), 0)
	s := stringSpan{&b[0], len(b) - 1}
	currentSpan := stringSpan{s.s, 0}
	currentSection := fieldInvalid
	currentField := fieldInvalid
	const (
		onNone = iota
		onKey
		onValue
		onComment
		onSection
	)
	state := onNone
	lenAtLastSpace := 0
	equalsLocation := 0
	for i := 0; i <= s.len; i++ {
		if i == s.len || *s.at(i) == '\n' || state != onComment && *s.at(i) == '#' {
			if state == onKey {
				currentSpan.len = lenAtLastSpace
				ret.append(s.s, currentSpan, highlightError)
			} else if state == onValue {
				if currentSpan.len != 0 {
					ret.append(s.s, stringSpan{s.at(equalsLocation), 1}, highlightDelimiter)
					currentSpan.len = lenAtLastSpace
					ret.highlightValue(s, currentSpan, currentField)
				} else {
					ret.append(s.s, stringSpan{s.at(equalsLocation), 1}, highlightError)
				}
			} else if state == onSection {
				currentSpan.len = lenAtLastSpace
				currentSection = currentSpan.sectionType()
				ret.append(s.s, currentSpan, validateHighlight(currentSection != fieldInvalid, highlightSection))
			} else if state == onComment {
				ret.append(s.s, currentSpan, highlightComment)
			}
			if i == s.len {
				break
			}
			lenAtLastSpace = 0
			currentField = fieldInvalid
			if *s.at(i) == '#' {
				currentSpan = stringSpan{s.at(i), 1}
				state = onComment
			} else {
				currentSpan = stringSpan{s.at(i + 1), 0}
				state = onNone
			}
		} else if state == onComment {
			currentSpan.len++
		} else if *s.at(i) == ' ' || *s.at(i) == '\t' {
			if s.at(i) == currentSpan.s && currentSpan.len == 0 {
				currentSpan.s = currentSpan.at(1)
			} else {
				currentSpan.len++
			}
		} else if *s.at(i) == '=' && state == onKey {
			currentSpan.len = lenAtLastSpace
			currentField = currentSpan.field()
			section := sectionForField(currentField)
			if section == fieldInvalid || currentField == fieldInvalid || section != currentSection {
				ret.append(s.s, currentSpan, highlightError)
			} else {
				ret.append(s.s, currentSpan, highlightField)
			}
			equalsLocation = i
			currentSpan = stringSpan{s.at(i + 1), 0}
			state = onValue
		} else {
			if state == onNone {
				if *s.at(i) == '[' {
					state = onSection
				} else {
					state = onKey
				}
			}
			currentSpan.len++
			lenAtLastSpace = currentSpan.len
		}
	}
	return ([]highlightSpan)(ret)
}

func highlightASecConfig(cfg string, spans []highlightSpan) {
	const (
		maxMTU  = 1500
		diffMTU = 80
	)

	var (
		mtu  = 0
		jc   = 0
		jmin = 0
		jmax = 0
		s1   = 0
		s2   = 0
		h1   = 0
		h2   = 0
		h3   = 0
		h4   = 0
	)

	var err error

	for i := range spans {
		span := &spans[i]
		switch span.t {
		case highlightError:
			return
		case highlightMTU:
			if mtu, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightJc:
			if jc, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightJmin:
			if jmin, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightJmax:
			if jmax, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightS1:
			if s1, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightS2:
			if s2, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightH1:
			if h1, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightH2:
			if h2, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightH3:
			if h3, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		case highlightH4:
			if h4, err = strconv.Atoi(cfg[span.s : span.s+span.len]); err != nil {
				return
			}
		}
	}

	if mtu == 0 {
		mtu = device.DefaultMTU
	}
	if h1 <= 4 {
		h1 = 1
	}
	if h2 <= 4 {
		h2 = 2
	}
	if h3 <= 4 {
		h3 = 3
	}
	if h4 <= 4 {
		h4 = 4
	}

	for i := range spans {
		span := &spans[i]
		switch span.t {
		case highlightJc:
			if jc > 128 {
				span.t = highlightWarning
			}
		case highlightJmin:
			if (jc != 0 || jmin != 0 || jmax != 0) && (jmin >= jmax || jmin >= mtu+diffMTU || jmin >= maxMTU) {
				span.t = highlightWarning
			}
		case highlightJmax:
			if (jc != 0 || jmin != 0 || jmax != 0) && (jmax <= jmin || jmax > mtu+diffMTU || jmax > maxMTU) {
				span.t = highlightWarning
			}
		case highlightS1:
			if s1+device.MessageInitiationSize == s2+device.MessageResponseSize {
				span.t = highlightError
			} else if s1 > mtu-device.MessageInitiationSize+diffMTU || s1 > maxMTU-device.MessageInitiationSize {
				span.t = highlightWarning
			}
		case highlightS2:
			if s1+device.MessageInitiationSize == s2+device.MessageResponseSize {
				span.t = highlightError
			} else if s2 > mtu-device.MessageResponseSize+diffMTU || s2 > maxMTU-device.MessageResponseSize {
				span.t = highlightWarning
			}
		case highlightH1:
			if h1 > 4 && (h1 == h2 || h1 == h3 || h1 == h4) {
				span.t = highlightError
			}
		case highlightH2:
			if h2 > 4 && (h2 == h1 || h2 == h3 || h2 == h4) {
				span.t = highlightError
			}
		case highlightH3:
			if h3 > 4 && (h3 == h1 || h3 == h2 || h3 == h4) {
				span.t = highlightError
			}
		case highlightH4:
			if h4 > 4 && (h4 == h1 || h4 == h2 || h4 == h3) {
				span.t = highlightError
			}
		}
	}
}
