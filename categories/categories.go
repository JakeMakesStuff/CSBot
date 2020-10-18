package categories

import "github.com/auttaja/gommand"

// Blackboard commands are used to allow for integration with the Blackboard service.
var Blackboard = &gommand.Category{
	Name:                 "Blackboard",
	Description:          "Blackboard commands are used to allow for integration with the Blackboard service.",
}

// Informational commands are used to return information to the user.
var Informational = &gommand.Category{
	Name:                 "Informational",
	Description:          "Informational commands are used to return information to the user.",
}

// Learning commands are commands which assist with learning.
var Learning = &gommand.Category{
	Name:                 "Learning",
	Description:          "Learning commands are commands which assist with learning.",
}
