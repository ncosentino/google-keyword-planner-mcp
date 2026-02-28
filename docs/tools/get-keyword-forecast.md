# get_keyword_forecast

Project future impressions, clicks, and cost for a set of keywords at a given maximum CPC bid. Useful for estimating the potential traffic from a paid campaign before committing ad spend.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `keywords` | string | Yes | Comma-separated keywords to forecast (e.g. `"dependency injection, SOLID principles"`) |
| `maxCpcMicros` | integer | No | Maximum CPC bid in micros. Default: `1000000` ($1.00). $0.50 = `500000`. |
| `forecastDays` | integer | No | Number of days to forecast. Default: `30`. |

## Response

Returns aggregate forecast data across all keywords:

```json
{
  "forecastDays": 30,
  "maxCpcMicros": 1000000,
  "impressions": 12500.0,
  "clicks": 350.0,
  "costMicros": 280000000,
  "clickThroughRate": 0.028,
  "averageCpcMicros": 800000
}
```

| Field | Description |
|-------|-------------|
| `forecastDays` | Number of days the forecast covers |
| `maxCpcMicros` | Max CPC bid used for the projection |
| `impressions` | Projected impressions |
| `clicks` | Projected clicks |
| `costMicros` | Projected total cost in micros ($1 = 1,000,000) |
| `clickThroughRate` | Projected click-through rate |
| `averageCpcMicros` | Projected average cost per click in micros |

## Example Prompts

- _"Forecast impressions and clicks for 'C# tutorial' and 'dotnet performance' at a $1.50 CPC bid over 30 days."_
- _"What would a $2 max CPC get me for the keyword 'software architecture'?"_
- _"Estimate the cost of running ads on these keywords for two weeks: dependency injection, SOLID principles, design patterns."_

## Notes

- `costMicros` divided by 1,000,000 gives the projected dollar cost.
- Forecasts are estimates; actual performance will vary based on auction competition and quality scores.
- This call requires a valid developer token with at least **basic access** -- test-mode tokens only work with test accounts.
