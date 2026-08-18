using System.Net.Http;
using Driftwatch;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var builder = Host.CreateApplicationBuilder(args);

builder.Services.Configure<DriftOptions>(
    builder.Configuration.GetSection(DriftOptions.SectionName));

// Allow the Teams webhook to come from an env var without putting it in config.
var envWebhook = Environment.GetEnvironmentVariable("DRIFTWATCH_TEAMS_WEBHOOK");
if (!string.IsNullOrWhiteSpace(envWebhook))
{
    builder.Services.PostConfigure<DriftOptions>(o => o.TeamsWebhookUrl = envWebhook);
}

builder.Services.AddHttpClient();
builder.Services.AddSingleton<TeamsReporter>(sp =>
{
    var httpFactory = sp.GetRequiredService<IHttpClientFactory>();
    var opts = sp.GetRequiredService<Microsoft.Extensions.Options.IOptions<DriftOptions>>().Value;
    return new TeamsReporter(
        httpFactory.CreateClient("teams"),
        sp.GetRequiredService<ILogger<TeamsReporter>>(),
        opts.TeamsWebhookUrl);
});

builder.Services.AddHostedService<DriftWorker>();

var host = builder.Build();
await host.RunAsync();
