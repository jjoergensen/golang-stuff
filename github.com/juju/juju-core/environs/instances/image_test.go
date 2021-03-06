// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instances

import (
	. "launchpad.net/gocheck"
	"launchpad.net/juju-core/constraints"
	"launchpad.net/juju-core/environs/imagemetadata"
	coretesting "launchpad.net/juju-core/testing"
	"testing"
)

type imageSuite struct {
	coretesting.LoggingSuite
}

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&imageSuite{})

func (s *imageSuite) SetUpSuite(c *C) {
	s.LoggingSuite.SetUpSuite(c)
}

func (s *imageSuite) TearDownSuite(c *C) {
	s.LoggingSuite.TearDownTest(c)
}

var jsonImagesContent = `
{
 "content_id": "com.ubuntu.cloud:released:aws",
 "products": {
   "com.ubuntu.cloud:server:12.04:amd64": {
     "release": "precise",
     "version": "12.04",
     "arch": "amd64",
     "versions": {
       "20121218": {
         "items": {
           "usee1pi": {
             "root_store": "instance",
             "virt": "pv",
             "region": "us-east-1",
             "id": "ami-00000011"
           },
           "usww1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "us-west-1",
             "id": "ami-00000016"
           },
           "apne1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "ap-northeast-1",
             "id": "ami-00000026"
           },
           "test1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "test",
             "id": "ami-00000033"
           },
           "test1he": {
             "root_store": "ebs",
             "virt": "hvm",
             "region": "test",
             "id": "ami-00000035"
           }
         },
         "pubname": "ubuntu-precise-12.04-amd64-server-20121218",
         "label": "release"
       },
       "20121118": {
         "items": {
           "apne1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "ap-northeast-1",
             "id": "ami-00000008"
           },
           "test2he": {
             "root_store": "ebs",
             "virt": "hvm",
             "region": "test",
             "id": "ami-00000036"
           }
         },
         "pubname": "ubuntu-precise-12.04-amd64-server-20121118",
         "label": "release"
       }
     }
   },
   "com.ubuntu.cloud:server:12.04:arm": {
     "release": "precise",
     "version": "12.04",
     "arch": "arm",
     "versions": {
       "20121218": {
         "items": {
           "apne1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "ap-northeast-1",
             "id": "ami-00000023"
           },
           "test1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "test",
             "id": "ami-00000034"
           },
           "armo1pe": {
             "root_store": "ebs",
             "virt": "pv",
             "region": "arm-only",
             "id": "ami-00000036"
           }
         },
         "pubname": "ubuntu-precise-12.04-arm-server-20121218",
         "label": "release"
       }
     }
   }
 },
 "format": "products:1.0"
}
`

type instanceSpecTestParams struct {
	desc             string
	region           string
	arches           []string
	constraints      string
	instanceTypes    []InstanceType
	imageId          string
	instanceTypeId   string
	instanceTypeName string
	err              string
}

func (p *instanceSpecTestParams) init() {
	if p.arches == nil {
		p.arches = []string{"amd64", "arm"}
	}
	if p.instanceTypes == nil {
		p.instanceTypes = []InstanceType{{Id: "1", Name: "it-1", Arches: []string{"amd64", "arm"}}}
		p.instanceTypeId = "1"
		p.instanceTypeName = "it-1"
	}
}

var pv = "pv"
var findInstanceSpecTests = []instanceSpecTestParams{
	{
		desc:    "image exists in metadata",
		region:  "test",
		imageId: "ami-00000033",
		instanceTypes: []InstanceType{
			{Id: "1", Name: "it-1", Arches: []string{"amd64"}, VType: &pv, Mem: 512},
		},
	},
	{
		desc:    "multiple images exists in metadata, use most recent",
		region:  "test",
		imageId: "ami-00000035",
		instanceTypes: []InstanceType{
			{Id: "1", Name: "it-1", Arches: []string{"amd64"}, VType: &hvm, Mem: 512, CpuCores: 2},
		},
	},
	{
		desc:   "no image exists in metadata",
		region: "invalid-region",
		err:    `no "precise" images in invalid-region with arches \[amd64 arm\]`,
	},
	{
		desc:          "no valid instance types",
		region:        "test",
		instanceTypes: []InstanceType{},
		err:           `no instance types in test matching constraints ""`,
	},
	{
		desc:          "no compatible instance types",
		region:        "arm-only",
		instanceTypes: []InstanceType{{Id: "1", Name: "it-1", Arches: []string{"amd64"}, Mem: 2048}},
		err:           `no "precise" images in arm-only matching instance types \[it-1\]`,
	},
}

func (s *imageSuite) TestFindInstanceSpec(c *C) {
	for _, t := range findInstanceSpecTests {
		c.Logf("test: %v", t.desc)
		t.init()
		ic := imagemetadata.ImageConstraint{
			CloudSpec: imagemetadata.CloudSpec{t.region, "ep"},
			Series:    "precise",
			Arches:    t.arches,
		}
		imageMeta, err := imagemetadata.GetLatestImageIdMetadata([]byte(jsonImagesContent), &ic)
		c.Assert(err, IsNil)
		var images []Image
		for _, imageMetadata := range imageMeta {
			im := *imageMetadata
			images = append(images, Image{
				Id:    im.Id,
				VType: im.VType,
				Arch:  im.Arch,
			})
		}
		spec, err := FindInstanceSpec(images, &InstanceConstraint{
			Series:      "precise",
			Region:      t.region,
			Arches:      t.arches,
			Constraints: constraints.MustParse(t.constraints),
		}, t.instanceTypes)
		if t.err != "" {
			c.Check(err, ErrorMatches, t.err)
			continue
		}
		if !c.Check(err, IsNil) {
			continue
		}
		c.Check(spec.Image.Id, Equals, t.imageId)
		if len(t.instanceTypes) == 1 {
			c.Check(spec.InstanceType, DeepEquals, t.instanceTypes[0])
		}
	}
}

var imageMatchtests = []struct {
	image Image
	itype InstanceType
	match bool
}{
	{
		image: Image{Arch: "amd64"},
		itype: InstanceType{Arches: []string{"amd64"}},
		match: true,
	}, {
		image: Image{Arch: "amd64"},
		itype: InstanceType{Arches: []string{"amd64", "arm"}},
		match: true,
	}, {
		image: Image{Arch: "amd64", VType: hvm},
		itype: InstanceType{Arches: []string{"amd64"}, VType: &hvm},
		match: true,
	}, {
		image: Image{Arch: "arm"},
		itype: InstanceType{Arches: []string{"amd64"}},
	}, {
		image: Image{Arch: "amd64", VType: hvm},
		itype: InstanceType{Arches: []string{"amd64"}},
		match: true,
	}, {
		image: Image{Arch: "amd64", VType: "pv"},
		itype: InstanceType{Arches: []string{"amd64"}, VType: &hvm},
	},
}

func (s *imageSuite) TestImageMatch(c *C) {
	for i, t := range imageMatchtests {
		c.Logf("test %d", i)
		c.Check(t.image.match(t.itype), Equals, t.match)
	}
}
