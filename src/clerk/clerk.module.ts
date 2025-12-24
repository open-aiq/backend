// clerk.module.ts
import { Module, Global } from '@nestjs/common';
import { createClerkClient } from '@clerk/backend';
import { CLERK_CLIENT } from 'src/app.consts';
import { AppConfigService } from 'src/config/config.service';

@Global()
@Module({
  providers: [
    {
      provide: CLERK_CLIENT,
      inject: [AppConfigService],

      useFactory: (config: AppConfigService) => {
        return createClerkClient({
          secretKey: config.getClerkSecretKey(),
          publishableKey: config.getClerkPublishableKey(),
        });
      },
    },
  ],
  exports: [CLERK_CLIENT],
})
export class ClerkModule {}
