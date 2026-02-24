// Package anilist provides a client for the Anilist GraphQL API.
package anilist

import "fmt"

// animeSubquery defines the common GraphQL selection set for anime metadata retrieval.
var animeSubquery = `
id
idMal
title {
	romaji
	english
	native
}
description(asHtml: false)
tags {
	name
	description
	rank
}
genres
coverImage {
	extraLarge
	large
	medium
	color
}
bannerImage
characters (page: 1, perPage: 10, role: MAIN) {
	nodes {
		id
		name {
			full
			native
		}
	}
}
startDate {
	year
	month	
	day
}
endDate {
	year
	month	
	day
}
staff {
	edges {
	  role
	  node {
		name {
		  full
		}
	  }
	}
}
status
synonyms
siteUrl
episodes
countryOfOrigin
externalLinks {
	url
}
averageScore
`

// searchByNameQuery defines the GraphQL query for searching anime by their title.
var searchByNameQuery = fmt.Sprintf(`
query ($query: String) {
	Page (page: 1, perPage: 30) {
		media (search: $query, type: ANIME) {
			%s
		}
	}
}
`, animeSubquery)

// searchByIDQuery defines the GraphQL query for retrieving a specific anime by its identifier.
var searchByIDQuery = fmt.Sprintf(`
query ($id: Int) {
	Media (id: $id, type: ANIME) {
		%s
	}
}`, animeSubquery)
