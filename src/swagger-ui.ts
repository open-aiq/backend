import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger';
import { INestApplication } from '@nestjs/common';

type PackageJson = {
  name: string;
  version: string;
  description: string;
};

// eslint-disable-next-line @typescript-eslint/no-require-imports
const packageJson: PackageJson = require('../package.json') as PackageJson;

export function setupSwagger(app: INestApplication): void {
  const config = new DocumentBuilder()
    .setTitle(packageJson.name)
    .setVersion(packageJson.version)
    .setDescription(packageJson.description)
    .addTag('App')
    .setContact(
      'Zeeshan Iqbal',
      'https://linkedin.com/in/zeeshan-Iqbal',
      'work.zeesh@gmail.com',
    )
    .build();

  const documentFactory = () => SwaggerModule.createDocument(app, config);
  SwaggerModule.setup('swagger-ui', app, documentFactory);
}
