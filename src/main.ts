import { AppModule } from './app.module';
import { setupSwagger } from './swagger-ui';
import { NestFactory } from '@nestjs/core';

async function bootstrap(): Promise<void> {
  const app = await NestFactory.create(AppModule);

  setupSwagger(app);

  await app.listen(process.env.PORT ?? 3000);
}

void bootstrap();
