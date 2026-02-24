-----------------------------------------------------------------------
-- AllAnime Scraper
--
-- Note: `http_tls` is a global injected by the Go Lua VM via
-- registerTLSClient(). It exposes:
--   http_tls.get(url [, headers_tbl]) → body (string)
--   http_tls.request({method, url, headers, body}) → {status, body}
-- Errors are raised as Lua errors, NOT returned as second values.
-----------------------------------------------------------------------

-- Constants
local AllanimeAPI   = "https://api.allanime.day/api"
local AllanimeBase  = "https://allanime.day"
local AllanimeRefr  = "https://allmanga.to"
local UA            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"

-- json is pre-loaded by mangal-lua-libs
local json = require("json")

-- Metadata
function Name()
    return "allanime"
end

function Base()
    return "https://allanime.to"
end

-- Decryption map for API responses.
-- Maps 2-char hex codes to decoded characters.
local decMap = {
    ["01"]="9", ["08"]="0", ["05"]="=", ["0a"]="2", ["0b"]="3",
    ["0c"]="4", ["07"]="?", ["00"]="8", ["5c"]="d", ["0f"]="7",
    ["5e"]="f", ["17"]="/", ["54"]="l", ["09"]="1", ["48"]="p",
    ["4f"]="w", ["0e"]="6", ["5b"]="c", ["5d"]="e", ["0d"]="5",
    ["53"]="k", ["1e"]="&", ["5a"]="b", ["59"]="a", ["4a"]="r",
    ["4c"]="t", ["4e"]="v", ["57"]="o", ["51"]="i",
    ["02"]=":", ["03"]=";", ["10"]="(", ["11"]=")", ["12"]="*",
    ["13"]="+", ["14"]=",", ["15"]="-", ["16"]=".", ["19"]="!",
    ["1b"]="#", ["1c"]="$", ["1d"]="%", ["46"]="~", ["63"]="[",
    ["65"]="]", ["67"]="_", ["78"]="@",
    -- Upper-case letters
    ["79"]="A", ["7a"]="B", ["7b"]="C", ["7c"]="D", ["7d"]="E",
    ["7e"]="F", ["7f"]="G", ["70"]="H", ["71"]="I", ["72"]="J",
    ["73"]="K", ["74"]="L", ["75"]="M", ["76"]="N", ["77"]="O",
    ["68"]="P", ["69"]="Q", ["6a"]="R", ["6b"]="S", ["6c"]="T",
    ["6d"]="U", ["6e"]="V", ["6f"]="W", ["60"]="X", ["61"]="Y",
    ["62"]="Z",
    -- Lower-case letters (remaining)
    ["50"]="h", ["52"]="j", ["55"]="m", ["56"]="n", ["58"]="q",
    ["49"]="q",  -- alternate
    ["4b"]="s", ["4d"]="u", ["47"]="x", ["40"]="x",  -- alternate
    ["41"]="y", ["42"]="z",
}

local function decrypt(hex)
    local result = ""
    for i = 1, #hex, 2 do
        local pair = string.sub(hex, i, i + 1)
        local ch = decMap[pair]
        if ch then
            result = result .. ch
        end
    end
    -- Append .json to /clock paths if needed
    result = string.gsub(result, "/clock$", "/clock.json")
    result = string.gsub(result, "/clock([^.])", "/clock.json%1")
    return result
end

-----------------------------------------------------------------------
-- Helper: URL Encode
-----------------------------------------------------------------------
local function urlencode(str)
    if str then
        str = string.gsub(str, "\n", "\r\n")
        str = string.gsub(str, "([^%w %-%_%.%~])",
            function(c) return string.format("%%%02X", string.byte(c)) end)
        str = string.gsub(str, " ", "+")
    end
    return str
end

-----------------------------------------------------------------------
-- Helper: make a GraphQL GET request to AllAnime API
-- Returns parsed JSON data or nil on error.
-----------------------------------------------------------------------
local function gqlRequest(gql, variables)
    local params = "variables=" .. urlencode(json.encode(variables)) .. "&query=" .. urlencode(gql)
    local fullUrl = AllanimeAPI .. "?" .. params

    -- http_tls.request returns a single table {status, body}
    -- Errors are raised as Lua errors (caught by pcall if needed)
    local reqOk, res = pcall(function()
        return http_tls.request({
            method  = "GET",
            url     = fullUrl,
            headers = {
                ["Referer"]    = AllanimeRefr,
                ["User-Agent"] = UA,
            },
        })
    end)

    if not reqOk or not res or res.status ~= 200 then
        -- error("Failed to request: " .. fullUrl)
        return nil
    end

    -- io.stderr:write("DEBUG BODY: " .. res.body .. "\n")
    local ok, data = pcall(json.decode, res.body)
    if not ok or not data then
        return nil
    end

    return data
end

-----------------------------------------------------------------------
-- Search
-----------------------------------------------------------------------
function SearchAnimes(query)
    -- io.stderr:write("DEBUG: SearchAnimes called with " .. query .. "\n")
    local gql = 'query( $search: SearchInput $limit: Int $page: Int $translationType: VaildTranslationTypeEnumType $countryOrigin: VaildCountryOriginEnumType ) { shows( search: $search limit: $limit page: $page translationType: $translationType countryOrigin: $countryOrigin ) { edges { _id name availableEpisodes __typename thumbnail } }}'

    local results = {}
    
    -- Fetch pages 1 to 3 to find more results (older shows like Code Geass TV often pushed back)
    for page = 1, 3 do
        local data = gqlRequest(gql, {
            search = {
                allowAdult = false,
                allowUnknown = false,
                query = query,
            },
            limit = 100, 
            page = page,
            translationType = "sub",
            countryOrigin = "ALL",
        })

        if data and data.data and data.data.shows and data.data.shows.edges then
            for _, show in ipairs(data.data.shows.edges) do
                local epCount = 0
                if show.availableEpisodes and show.availableEpisodes.sub then
                    epCount = tonumber(show.availableEpisodes.sub) or 0
                end
                
                table.insert(results, {
                    name  = show.name .. " (" .. epCount .. " eps)",
                    url   = show._id,
                    cover = show.thumbnail,
                    epCount = epCount, -- Helper for sorting
                })
            end
        end
    end

    -- Sort by episode count descending (Main series usually have more eps than movies/OVAs)
    table.sort(results, function(a, b)
        return a.epCount > b.epCount
    end)

    return results
end

-----------------------------------------------------------------------
-- Episodes
-----------------------------------------------------------------------
function AnimeEpisodes(anime_data)
    local showId
    if type(anime_data) == "table" then
        showId = anime_data.url
    else
        showId = anime_data
    end
    -- io.stderr:write("DEBUG: AnimeEpisodes called with " .. tostring(showId) .. "\n")

    local gql = 'query ($showId: String!) { show( _id: $showId ) { _id availableEpisodesDetail }}'

    local data = gqlRequest(gql, { showId = showId })

    if not data or not data.data or not data.data.show then
        return {}
    end

    local details = data.data.show.availableEpisodesDetail.sub
    local episodes = {}

    if details then
        for _, epStr in ipairs(details) do
            table.insert(episodes, {
                name   = "Episode " .. epStr,
                url    = showId .. ":" .. epStr,
                number = tonumber(epStr) or 0,
            })
        end

        -- FORCE ASCENDING SORT
        table.sort(episodes, function(a, b)
            return a.number < b.number
        end)
    end

    return episodes
end

-----------------------------------------------------------------------
-- Stream Extraction (ChapterPages)
-----------------------------------------------------------------------

-- Priority order for stream providers.
local providerPriority = { "Luf-Mp4", "Default", "S-mp4", "Yt-mp4", "Sak", "Kir" }

function EpisodeVideos(episode_url)
    -- If input is table (from custom provider), extract url field
    if type(episode_url) == "table" then
        episode_url = episode_url.url
    end

    -- url format: showId:episodeString
    local parts = {}
    for part in string.gmatch(episode_url, "([^:]+)") do
        table.insert(parts, part)
    end
    local showId = parts[1]
    local epStr  = parts[2]

    local gql = 'query ($showId: String!, $translationType: VaildTranslationTypeEnumType!, $episodeString: String!) { episode( showId: $showId translationType: $translationType episodeString: $episodeString ) { episodeString sourceUrls }}'

    local data = gqlRequest(gql, {
        showId = showId,
        translationType = "sub",
        episodeString = epStr,
    })

    if not data or not data.data or not data.data.episode then
        return {}
    end

    local sourceUrls = data.data.episode.sourceUrls
    if not sourceUrls then
        return {}
    end

    -- Build a map of sourceName → decrypted path
    local sourceMap = {}
    for _, source in ipairs(sourceUrls) do
        local sName = source.sourceName or ""
        local sUrl  = source.sourceUrl or ""
        if string.sub(sUrl, 1, 2) == "--" then
            local hex = string.sub(sUrl, 3)
            local path = decrypt(hex)
            if path and path ~= "" then
                sourceMap[sName] = path
            end
        end
    end

    -- Try providers in priority order
    for _, provName in ipairs(providerPriority) do
        local path = sourceMap[provName]
        if path then
            local result = fetchProvider(path)
            if result then
                return result
            end
        end
    end

    -- Fallback: try any decrypted source
    for _, path in pairs(sourceMap) do
        if path then
            local result = fetchProvider(path)
            if result then
                return result
            end
        end
    end

    return {}
end

-----------------------------------------------------------------------
-- Helper: Resolve Embed (Extract stream from hosting sites)
-----------------------------------------------------------------------
local function resolve_embed(url)
    -- s3taku / gogoplay / anitaku
    if string.find(url, "s3taku") or string.find(url, "gogoplay") or string.find(url, "anitaku") or string.find(url, "gotaku1") then
        local res = http_tls.request({ method = "GET", url = url })
        if res and res.body then
            -- Try to find file: '...' in JWPlayer config
            local link = string.match(res.body, "file:%s*['\"](https?://[^'\"]+)['\"]")
            return link or url
        end
    end
    
    return url
end

-----------------------------------------------------------------------
-- fetchProvider: given a decrypted path, fetch the actual video link.
-- The decrypted path is an endpoint on AllanimeBase.
-- It typically returns JSON with a `links` array.
-----------------------------------------------------------------------
function fetchProvider(path)
    local embedUrl
    if string.sub(path, 1, 4) == "http" then
        embedUrl = path
    else
        embedUrl = AllanimeBase .. path
    end

    -- Use pcall because some providers may 404 or return garbage
    local ok, res = pcall(function()
        return http_tls.request({
            method  = "GET",
            url     = embedUrl,
            headers = {
                ["Referer"]    = AllanimeRefr,
                ["User-Agent"] = UA,
            },
        })
    end)

    if not ok or not res or not res.body then
        return nil
    end

    -- Try to parse as JSON
    local parseOk, embedData = pcall(json.decode, res.body)
    if parseOk and embedData and embedData.links and #embedData.links > 0 then
        -- Find best quality link
        local bestLink = nil
        local bestRes  = 0
        local bestIsHls = false

        for _, l in ipairs(embedData.links) do
            if l.link and l.link ~= "" then
                 -- Check for HLS/m3u8 preference
                local isHls = string.find(l.link, ".m3u8")
                local resNum = tonumber(l.resolutionStr) or 0
                
                -- Prefer HLS if resolution is similar or better, or if we haven't found anything yet
                if (resNum > bestRes) or (isHls and not bestIsHls and resNum >= bestRes) then
                    bestRes  = resNum
                    bestLink = l.link
                    bestIsHls = isHls
                end
            end
        end

        -- Fallback to first link if no resolution info
        if not bestLink then
            bestLink = embedData.links[1].link
        end

        if bestLink and bestLink ~= "" then
            -- RESOLVE EMBED HERE
            local finalLink = resolve_embed(bestLink)
            
            return {{
                url = finalLink,
                headers = {
                    ["Referer"] = AllanimeRefr,
                    ["User-Agent"] = UA,
                },
            }}
        end
    end

    -- If not JSON or no links, maybe the URL itself is a direct stream
    if string.find(embedUrl, "mp4") or string.find(embedUrl, "m3u8") then
        return {{
            url = embedUrl,
            headers = {
                ["Referer"] = AllanimeRefr,
                ["User-Agent"] = UA,
            },
        }}
    end

    return nil
end






