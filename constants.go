package virtualbox

var Linux32 OSType = OSType{ID: "Linux", FamilyID: "Linux", Bit64: false}
var Linux64 OSType = OSType{ID: "Linux_64", FamilyID: "Linux", Bit64: true}

var Ubuntu32 OSType = OSType{ID: "Ubuntu", FamilyID: "Linux", Bit64: false}
var Ubuntu64 OSType = OSType{ID: "Ubuntu_64", FamilyID: "Linux", Bit64: true}
