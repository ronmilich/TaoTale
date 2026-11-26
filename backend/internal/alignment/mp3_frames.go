package alignment

import (
	"bytes"
	"encoding/binary"
)

// FFmpeg can conceal a partial last frame. Check MPEG Layer III frame lengths
// before decoding, and honor the declared stream length when Xing/Info has one.
func completeFrames(data []byte) bool {
	if bytes.HasPrefix(data, []byte("ID3")) {
		if len(data) < 10 || (data[3] != 2 && data[3] != 3 && data[3] != 4) ||
			(data[6]|data[7]|data[8]|data[9])&0x80 != 0 {
			return false
		}
		size := 10 + (int(data[6])<<21 | int(data[7])<<14 | int(data[8])<<7 | int(data[9]))
		if data[3] == 4 && data[5]&0x10 != 0 {
			size += 10
		}
		if size > len(data) {
			return false
		}
		data = data[size:]
	}
	total := len(data)
	if total >= 128 && bytes.Equal(data[total-128:total-125], []byte("TAG")) {
		data = data[:total-128]
	}
	declared, frames := uint32(0), uint32(0)
	for len(data) > 0 {
		size, sideInfo := frameSize(data)
		if size == 0 || size > len(data) {
			return false
		}
		if frames == 0 && size >= sideInfo+12 {
			header := string(data[sideInfo : sideInfo+4])
			if (header == "Xing" || header == "Info") && binary.BigEndian.Uint32(data[sideInfo+4:sideInfo+8])&1 != 0 {
				declared = binary.BigEndian.Uint32(data[sideInfo+8 : sideInfo+12])
			}
		}
		frames++
		data = data[size:]
	}
	return frames > 0 && (declared == 0 || frames-1 == declared)
}

func frameSize(data []byte) (size, sideInfo int) {
	if len(data) < 4 || data[0] != 0xff || data[1]&0xe0 != 0xe0 || data[1]&6 != 2 {
		return 0, 0
	}
	version, bitrateIndex, sampleIndex := (data[1]>>3)&3, data[2]>>4, (data[2]>>2)&3
	if version == 1 || sampleIndex == 3 || bitrateIndex == 0 || bitrateIndex == 15 {
		return 0, 0
	}
	rate := [...]int{44100, 48000, 32000}[sampleIndex]
	factor, bitrate := 72, [...]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}[bitrateIndex]
	mono := data[3]&0xc0 == 0xc0
	sideInfo = 21
	if mono {
		sideInfo = 13
	}
	switch version {
	case 0:
		rate /= 4
	case 2:
		rate /= 2
	case 3:
		factor, bitrate = 144, [...]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}[bitrateIndex]
		sideInfo = 36
		if mono {
			sideInfo = 21
		}
	}
	if data[1]&1 == 0 {
		sideInfo += 2
	}
	return factor*bitrate*1000/rate + int((data[2]>>1)&1), sideInfo
}
