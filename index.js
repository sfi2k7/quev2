import { serve } from "https://deno.land/std@0.97.0/http/server.ts";
let server = serve({ port: 9870 });
for await (const req of server) {
  console.log("Request Arrived", req);
  req.respond({ body: "Hello Deno" });
}
