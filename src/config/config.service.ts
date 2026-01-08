import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { CLERK_SECRET_KEY, CLERK_PUBLISHABLE_KEY } from 'src/app.consts';

@Injectable()
export class AppConfigService {
  constructor(private readonly configService: ConfigService) {
    const v = configService.getOrThrow(null, { infer: true });

    console.log(v);
  }

  getClerkSecretKey(): string {
    return this.configService.getOrThrow<string>(CLERK_SECRET_KEY);
  }

  getClerkPublishableKey(): string {
    return this.configService.getOrThrow<string>(CLERK_PUBLISHABLE_KEY);
  }
}
