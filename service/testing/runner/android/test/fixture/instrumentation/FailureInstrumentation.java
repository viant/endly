package com.viant.endly.failuretest;

import android.app.Activity;
import android.app.Instrumentation;
import android.os.Bundle;

public final class FailureInstrumentation extends Instrumentation {
    @Override
    public void onCreate(Bundle arguments) {
        super.onCreate(arguments);
        start();
    }

    @Override
    public void onStart() {
        Bundle started = new Bundle();
        started.putString("class", "com.viant.endly.failuretest.FailureInstrumentation");
        started.putString("test", "intentionalFailure");
        sendStatus(1, started);

        Bundle failed = new Bundle(started);
        failed.putString("stack", "java.lang.AssertionError: intentional Endly instrumentation failure");
        sendStatus(-2, failed);

        Bundle result = new Bundle();
        result.putString("shortMsg", "intentional Endly instrumentation failure");
        finish(Activity.RESULT_CANCELED, result);
    }
}
