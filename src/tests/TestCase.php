<?php

namespace Tests;

use Illuminate\Foundation\Testing\TestCase as BaseTestCase;
use Illuminate\Support\Facades\DB;
use RuntimeException;

abstract class TestCase extends BaseTestCase
{
    /**
     * Tests rebuild their schema on every run. Refuse to touch anything but the
     * in-memory SQLite database, so a stray DB_* environment variable (Docker
     * exports one) can never point the suite at a real database.
     */
    protected function setUp(): void
    {
        parent::setUp();

        $connection = DB::connection();

        if ($connection->getDriverName() !== 'sqlite' || $connection->getDatabaseName() !== ':memory:') {
            throw new RuntimeException(sprintf(
                'Refusing to run tests against the "%s" connection (%s). Expected in-memory SQLite — check phpunit.xml and the DB_* environment.',
                $connection->getName(),
                $connection->getDatabaseName(),
            ));
        }
    }
}
