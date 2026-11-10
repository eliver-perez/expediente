<?php
declare(strict_types=1);
namespace Aibid\Tests\System;

use Aibid\{App, Config};
use Aibid\Application\Actor;
use Aibid\Domain\{Protocol, Uuid};
use Aibid\Tests\{MySqlFixture, NativeStackFixture, RealClientBridge};
use PHPUnit\Framework\TestCase;

/** Solo se copia a una instancia temporal: TLS/FPM/MySQL y cliente Go reales. */
final class LicenseReviewTest extends TestCase
{
    public function testOfflineReplacementAndAuthenticatedRevocationWithRealClient(): void
    {
        $fixture = new MySqlFixture();
        $stack = null; $bridge = null;
        try {
            $stack = new NativeStackFixture($fixture);
            $owner = new App(new Config($stack->values, false), $fixture->owner);
            $owner->signingKeys->generate('qa-random-license', 'license');
            $owner->signingKeys->activate('qa-random-license', 'Validación sintética aislada');
            $keys = array_map(fn($key) => ['kid'=>$key['kid'],'public_key'=>Protocol::encode($key['public_key']),'environment'=>'production','purpose'=>'license'], $fixture->owner->all("SELECT kid,public_key FROM signing_keys WHERE purpose='license'"));
            $app = $stack->app;
            $actor = new Actor($app->auth->bootstrap('qa@example.test', 'QA sintético', 'Synthetic-Only-2026!'), 'superadmin');
            $customer = $app->catalog->saveCustomer($actor, Uuid::create(), null, ['name'=>'QA aislado','contact_name'=>'Sintético','email'=>'','state'=>'active']);
            $license = $app->licenses->issue($actor, Uuid::create(), ['customer_id'=>$customer['id'],'product_id'=>'gestor_documental','license_type'=>'perpetual','entitled_release_until'=>gmdate('Y-m-d\TH:i'),'features'=>['linked_libraries','managed_libraries','ocr','expedientes','review_workflow'],'reason'=>'QA sin licencias comerciales']);
            $bridge = new RealClientBridge(getenv('TEST_CLIENT_SOURCE'));
            $call = fn(string $installation, string $operation, array $input = []) => $bridge->run($stack, $installation, $keys, ['operation'=>$operation]+$input);
            // Los nombres representan las instalaciones; ambos procesos Go se ejecutan en macOS.
            $first = $call('origen', 'online', ['action'=>'activate','request_id'=>Uuid::create(),'commercial_key'=>$license['secret']]);
            self::assertArrayNotHasKey('error', $first); self::assertTrue($first['write_allowed']);
            $sentinel = $bridge->directory.'/state-origen/documento-sintetico.txt';
            file_put_contents($sentinel, 'Original sintético intacto');
            $before = hash_file('sha256', $sentinel);
            $request = $call('destino', 'offline', ['action'=>'activate','request_id'=>Uuid::create()]);
            self::assertArrayNotHasKey('error', $request);
            $pending = $app->offline->import($actor, Uuid::create(), $request['contents']);
            $decision = $app->offline->decide($actor, Uuid::create(), $pending['id'], 'approved', ['license_id'=>$license['id'],'version'=>$app->read->license($license['id'])['row_version'],'renewal_mode'=>'current','force_activation_id'=>$first['status']['activation_id'],'accept_offline_limit'=>'1','reason'=>'Sustitución offline sintética']);
            $jws = $app->read->revision($decision['license_id'], $decision['revision_id'])['license_jws'];
            $second = $call('destino', 'import', ['contents'=>$jws]);
            self::assertArrayNotHasKey('error', $second); self::assertTrue($second['write_allowed']);
            self::assertSame(1, (int)$fixture->owner->execute("SELECT count(*) FROM activations WHERE state='active'")->fetchColumn());
            self::assertSame('revoked', $fixture->owner->execute('SELECT state FROM activations WHERE activation_id=?', [Uuid::bytes($first['status']['activation_id'])])->fetchColumn());
            // El origen aún no conoce el retiro: no existe revocación instantánea offline.
            self::assertTrue($call('origen', 'status')['write_allowed']);
            $terminal = $call('origen', 'online', ['action'=>'refresh','request_id'=>Uuid::create()]);
            self::assertArrayNotHasKey('error', $terminal); self::assertSame('revoked', $terminal['status']['state']);
            self::assertFalse($terminal['write_allowed']); self::assertTrue($terminal['read_allowed']); self::assertTrue($terminal['export_allowed']); self::assertFalse($terminal['ocr_allowed']);
            // Cada invocación del bridge abre otra vez identidad y base persistidas.
            self::assertSame('revoked', $call('origen', 'status')['status']['state']);
            self::assertSame($before, hash_file('sha256', $sentinel));
            $lost = $call('destino', 'online', ['origin'=>'https://127.0.0.1:1','action'=>'refresh','request_id'=>Uuid::create()]);
            self::assertSame('TEMPORARY_UNAVAILABLE', $lost['error']); self::assertTrue($lost['write_allowed']);
            self::assertSame('active', $call('destino', 'status')['status']['state']);
            $restored = $call('destino', 'online', ['action'=>'refresh','request_id'=>Uuid::create()]);
            self::assertArrayNotHasKey('error', $restored); self::assertTrue($restored['write_allowed']);
            $app->licenses->change($actor, Uuid::create(), $license['id'], 'revoke', ['version'=>$app->read->license($license['id'])['row_version'],'reason'=>'Revocación comercial sintética']);
            $revoked = $call('destino', 'online', ['action'=>'refresh','request_id'=>Uuid::create()]);
            self::assertArrayNotHasKey('error', $revoked); self::assertSame('revoked', $revoked['status']['state']); self::assertFalse($revoked['write_allowed']); self::assertTrue($revoked['read_allowed']);
            self::assertSame('revoked', $call('destino', 'status')['status']['state']);
            self::assertSame($before, hash_file('sha256', $sentinel));
            self::assertGreaterThan(0, $app->audit->verify());
        } finally {
            if ($bridge !== null) { $bridge->destroy(); }
            if ($stack !== null) { $stack->destroy(); }
            $fixture->destroy();
        }
    }
}
