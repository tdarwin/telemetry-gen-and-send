Okay, what I would like to build here is a system for generating OpenTelemetry traces, logs, and metrics, as well as for sending those metrics in high volumes to an OTLP endpoint.  This should be at least two golang tools, one for generation and one for sending, though we could have different generators for logs metrics and traces if that's easier.

What I'm imagining is that the generator is creating templates that the sender can load into memory to multiply the output and make the generated telemetry look "current" while keeping a similar shape every time I run the sender.

Ultimately, we're trying to be able to scale up load generation to 15 million events per second (where an event is a span, a log line, or a metric name and timeseries).  It may be that we run the generator and then have to let multiple senders load that data in order to test it, and that's fine. It'd be ideal if this was as resource efficient as possible to run.

As we develop this together, ask me for input on implementation if not sure of the best way to implement something.

## Telemetry Generator

We want to generate logs, metrics and traces in an OTLP format, and save them to files that can then be used by the sender.  The only thing of note about the telemetry generated is that it should not have timestamps.  Spans should have a duration, but we want to generate repeatable telemetry that the sender can duplicate and send multiple times with new timestamps and trace/span IDs so that we're able to send a lot more traffic than we generate.

### Traces

In generating traces we want to be able to do the following things
 - specify the number of spans on average in a trace (actual output should have a variable number of spans where the average is near the number specified)
 - allow for the creation of very-high spancount traces.  We want to be able to send the occasional trace with hundreds of thousands of spans, so we should be able to generate that.
 - we want to use OpenTelemetry symantec conventions in our trace data. We should have attributes for HTTP and Database calls.  Use the context7 MCP to determine if we 
 - We should be able to give a number of services that the traces should move through in their spans.
 - We should be able to say that there is one ingress service or multiple ingresses to our service map (this basically determines if our traces all start with one service or if the root span of the traces contains multiple services)
 - we should be able to specify a number of custom attributes we want randomly added to spans in the trace.  The generated attribute names should remain consistent, and values for each attribute should stick to the same type (int, string, float, boolean) though they can be different values for each use of an attribute on a span.

### Metrics

We want to generate metrics that allows for our sender to send the following:
  - We should use OpenTelemetry host metrics and kubernetes cluster, node, container, and pod metrics as examples for generating metrics
  - We should be able to specify the number of metric names we want to generate, and should be able to generate up to 2,000 metric names.
  - Each metric name should have hundreds of time series per metric
  - We will want to make sure each time series will persist for the length of the run by the sender.

### Logs

We want to generate log messages in OTLP log format, again with timestamps that can be updated by the sender so that we can send these messages repeatedly.  Use the following when designing the generator for logs:
  - Should have multiple types of logs: HTTP access logs, application logs, and system logs
  - Should have different service names for the application logs and should be able to specify the number of "services"

  
## Telemetry Sender

This should ingest the telemetry created by the generator and do the following:
  - update any timestamps to be current (some event latency is fine, but a "last 10 minute" window in my Honeycomb query should show new events while the sender is running)
  - update trace and span IDs in trace data so that we're not creating duplicate events with updated dates.
  - add timeseries to generated metric names so that increasing the volume of datapoints to 20k over a 1 minute window is possible.
