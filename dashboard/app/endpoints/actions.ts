"use server";

import { revalidatePath } from "next/cache";
import {
  createEndpoint,
  updateEndpoint,
  deleteEndpoint,
  rotateSecret,
  createSubscription,
  deleteSubscription,
} from "@/lib/api";
import type { Endpoint } from "@/lib/types";

type Result<T = unknown> =
  | ({ ok: true } & T)
  | { ok: false; error: string };

function fail(e: unknown): { ok: false; error: string } {
  return { ok: false, error: e instanceof Error ? e.message : "request failed" };
}

export async function createEndpointAction(input: {
  name: string;
  url: string;
  timeout_ms?: number;
  max_retries?: number;
}): Promise<Result<{ id: string; secret: string }>> {
  try {
    const ep = await createEndpoint(input);
    revalidatePath("/endpoints");
    // Secret is returned by the API exactly once, at creation.
    return { ok: true, id: ep.id, secret: ep.secret };
  } catch (e) {
    return fail(e);
  }
}

// Full overwrite — the API's UPDATE replaces every column, so resend all
// current fields and only change `enabled`. Sending a partial body would
// blank the name/url.
export async function setEnabledAction(
  ep: Endpoint,
  enabled: boolean
): Promise<Result> {
  try {
    await updateEndpoint(ep.id, {
      name: ep.name,
      url: ep.url,
      enabled,
      timeout_ms: ep.timeout_ms,
      max_retries: ep.max_retries,
    });
    revalidatePath("/endpoints");
    return { ok: true };
  } catch (e) {
    return fail(e);
  }
}

export async function deleteEndpointAction(id: string): Promise<Result> {
  try {
    const res = await deleteEndpoint(id);
    if (!res.ok) throw new Error(`${res.status}`);
    revalidatePath("/endpoints");
    return { ok: true };
  } catch (e) {
    return fail(e);
  }
}

export async function rotateSecretAction(
  id: string
): Promise<Result<{ secret: string }>> {
  try {
    const { secret } = await rotateSecret(id);
    revalidatePath("/endpoints");
    return { ok: true, secret };
  } catch (e) {
    return fail(e);
  }
}

export async function addSubscriptionAction(
  endpointID: string,
  eventTypes: string[]
): Promise<Result> {
  try {
    await createSubscription(endpointID, eventTypes);
    revalidatePath("/endpoints");
    return { ok: true };
  } catch (e) {
    return fail(e);
  }
}

export async function deleteSubscriptionAction(
  endpointID: string,
  subID: string
): Promise<Result> {
  try {
    const res = await deleteSubscription(endpointID, subID);
    if (!res.ok) throw new Error(`${res.status}`);
    revalidatePath("/endpoints");
    return { ok: true };
  } catch (e) {
    return fail(e);
  }
}
