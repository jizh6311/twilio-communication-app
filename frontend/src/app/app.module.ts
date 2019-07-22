import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { TextSenderComponent } from './components/textSender/textSender.component';
import { VoiceSenderComponent } from './components/voiceSender/voiceSender.component';

@NgModule({
  declarations: [
    AppComponent,
    TextSenderComponent,
    VoiceSenderComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
