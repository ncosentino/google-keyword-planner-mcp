using System.Globalization;
using System.Net;
using System.Text.Json;
using KeywordPlannerMcp.KeywordPlanner;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class ForecastConcurrencyTests
{
    private sealed class Pacer : IPlanningPacer
    {
        private int _calls;
        public int Calls => _calls;
        public Task WaitAsync(CancellationToken cancellationToken)
        {
            cancellationToken.ThrowIfCancellationRequested();
            Interlocked.Increment(ref _calls);
            return Task.CompletedTask;
        }
    }

    private sealed class Handler : HttpMessageHandler
    {
        private int _active;
        private int _started;
        private int _maximumActive;
        private int _cancelled;
        private int _combined;
        public int MaximumActive => _maximumActive;
        public int Started => _started;
        public int Cancelled => _cancelled;
        public int Combined => _combined;
        public TaskCompletionSource AllEntered { get; } = new(TaskCreationOptions.RunContinuationsAsynchronously);
        public TaskCompletionSource[] Release { get; } = Enumerable.Range(0, 5)
            .Select(_ => new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously)).ToArray();
        public TaskCompletionSource[] Completed { get; } = Enumerable.Range(0, 5)
            .Select(_ => new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously)).ToArray();
        public bool FailFirst { get; set; }

        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            if (request.RequestUri!.Host == "oauth2.googleapis.com")
                return Response("""{"access_token":"fake","expires_in":3600}""");
            using var json = JsonDocument.Parse(await request.Content!.ReadAsStringAsync(cancellationToken));
            var keywords = json.RootElement.GetProperty("campaign").GetProperty("adGroups")[0].GetProperty("keywords");
            if (keywords.GetArrayLength() > 1)
            {
                Interlocked.Increment(ref _combined);
                return Response("""{"campaignForecastMetrics":{"clicks":999}}""");
            }
            var index = int.Parse(keywords[0].GetProperty("text").GetString()!, CultureInfo.InvariantCulture);
            var active = Interlocked.Increment(ref _active);
            int previous;
            do { previous = _maximumActive; }
            while (active > previous && Interlocked.CompareExchange(ref _maximumActive, active, previous) != previous);
            if (Interlocked.Increment(ref _started) == 5) AllEntered.TrySetResult();
            try
            {
                if (index < 5) await Release[index].Task.WaitAsync(cancellationToken);
                if (index == 0 && FailFirst) return Response("first failure", HttpStatusCode.BadRequest);
                return Response("{\"campaignForecastMetrics\":{\"clicks\":"
                    + index.ToString(CultureInfo.InvariantCulture) + "}}");
            }
            catch (OperationCanceledException)
            {
                Interlocked.Increment(ref _cancelled);
                throw;
            }
            finally
            {
                Interlocked.Decrement(ref _active);
                if (index < 5) Completed[index].TrySetResult();
            }
        }
    }

    private static HttpResponseMessage Response(string body, HttpStatusCode status = HttpStatusCode.OK) =>
        new(status) { Content = new StringContent(body) };

    private static KeywordPlannerClient Client(Handler handler, Pacer pacer)
    {
        var http = new HttpClient(handler);
        return new KeywordPlannerClient("dev", "123", null,
            new OAuth2TokenProvider("id", "secret", "refresh", http), http, planningPacer: pacer);
    }

    [Fact]
    public async Task Forecast_BoundedWorkersPreserveInputOrderDespiteReverseCompletion()
    {
        using var handler = new Handler();
        var pacer = new Pacer();
        using var client = Client(handler, pacer);
        var keywords = Enumerable.Range(0, 8).Select(i => i.ToString(CultureInfo.InvariantCulture)).ToArray();
        var resultTask = client.GetKeywordForecastAsync(keywords, 1, 30, includeKeywordBreakdown: true);
        await handler.AllEntered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        Assert.Equal(5, handler.Started);
        Assert.Equal(5, handler.MaximumActive);
        for (var i = 4; i >= 0; i--)
        {
            handler.Release[i].TrySetResult();
            await handler.Completed[i].Task.WaitAsync(TimeSpan.FromSeconds(5));
        }
        var result = await resultTask.WaitAsync(TimeSpan.FromSeconds(5));
        Assert.Equal(keywords, result.Keywords.Select(k => k.Text));
        Assert.Equal(Enumerable.Range(0, 8).Select(i => (double?)i), result.Keywords.Select(k => k.Clicks));
        Assert.Equal(999d, result.Total.Clicks);
        Assert.Equal(1, handler.Combined);
        Assert.Equal(9, pacer.Calls);
        Assert.Equal(5, handler.MaximumActive);
    }

    [Fact]
    public async Task Forecast_FirstApiFailureCancelsSiblingsAndDoesNotRequestCombinedTotal()
    {
        using var handler = new Handler { FailFirst = true };
        using var client = Client(handler, new Pacer());
        var resultTask = client.GetKeywordForecastAsync(["0", "1", "2", "3", "4", "5"], 1, 30,
            includeKeywordBreakdown: true);
        await handler.AllEntered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        handler.Release[0].TrySetResult();
        var failure = await Assert.ThrowsAsync<InvalidOperationException>(() => resultTask);
        Assert.Contains("first failure", failure.Message);
        Assert.Equal(4, handler.Cancelled);
        Assert.Equal(5, handler.Started);
        Assert.Equal(0, handler.Combined);
    }

    [Fact]
    public async Task Forecast_CallerCancellationCancelsEveryActiveScenarioAndPropagates()
    {
        using var handler = new Handler();
        using var client = Client(handler, new Pacer());
        using var cancellation = new CancellationTokenSource();
        var resultTask = client.GetKeywordForecastAsync(["0", "1", "2", "3", "4", "5"], 1, 30,
            cancellation.Token, includeKeywordBreakdown: true);
        await handler.AllEntered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => resultTask);
        Assert.Equal(5, handler.Cancelled);
        Assert.Equal(5, handler.Started);
        Assert.Equal(0, handler.Combined);
    }
}
