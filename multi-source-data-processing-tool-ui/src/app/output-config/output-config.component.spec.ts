import { ComponentFixture, TestBed } from '@angular/core/testing';

import { OutputConfigComponent } from './output-config.component';

describe('OutputConfigComponent', () => {
  let component: OutputConfigComponent;
  let fixture: ComponentFixture<OutputConfigComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [OutputConfigComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(OutputConfigComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
