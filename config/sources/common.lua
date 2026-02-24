-- HTTP header logic for scrapers
local common = {}

-- agents registry of common User-Agent strings to mitigate automated scraping detection.
local agents = {
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64; rv:122.0) Gecko/20100101 Firefox/122.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15"
}

-- get_headers returns a table of HTTP headers structured for standard browser emulation.
function common.get_headers(referer)
    return {
        ["User-Agent"] = agents[math.random(#agents)],
        ["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
        ["Accept-Language"] = "en-US,en;q=0.5",
        ["Referer"] = referer or "https://google.com",
        ["DNT"] = "1",
        ["Sec-Fetch-Dest"] = "document",
        ["Sec-Fetch-Mode"] = "navigate"
    }
end

return common
